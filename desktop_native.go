//go:build desktop

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Independent implementation using the MIT-licensed Wails framework. No
// MaClaw source or pet artwork is incorporated.
func init() { runDesktop = startDesktop }
func startDesktop(a *App, link string) error {
	dir := filepath.Dir(a.profiles.path)
	cache := filepath.Join(dir, "teaching-agent-runtime", "webview")
	if e := os.MkdirAll(cache, 0700); e != nil {
		return fmt.Errorf("无法创建原生窗口缓存目录：%w", e)
	}
	windows := application.WindowsOptions{WebviewUserDataPath: cache}
	// Native QA only: do not expose a debugging port in normal installations.
	if os.Getenv("TEACHER_QA_DIR") != "" {
		windows.AdditionalBrowserArgs = []string{"--remote-debugging-port=9222"}
	}
	host := application.New(application.Options{
		Name: "教学创新备赛工作台", Description: "报告、课堂实录与教学汇报审阅",
		Mac:     application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		Windows: windows,
	})
	var mu sync.Mutex
	var pet *application.WebviewWindow
	var workbench *application.WebviewWindow
	var dragAnchor struct {
		active                             bool
		screenX, screenY, windowX, windowY int
	}
	count := 0
	var visible atomic.Bool
	open := func(windowURL string) {
		count++
		g, ok := a.loadGeometry("workbench")
		if !ok {
			g = windowGeometry{Width: 1440, Height: 860, X: 80, Y: 60}
		}
		screen := host.Screen.GetPrimary()
		for _, candidate := range host.Screen.GetAll() {
			r := candidate.WorkArea
			if g.X >= r.X && g.X < r.X+r.Width && g.Y >= r.Y && g.Y < r.Y+r.Height {
				screen = candidate
				break
			}
		}
		if screen != nil {
			r := screen.WorkArea
			g = clampGeometry(g, r.X, r.Y, r.Width, r.Height)
		}
		w := host.Window.NewWithOptions(application.WebviewWindowOptions{Name: fmt.Sprintf("workbench-%d", count), Title: "教学创新备赛工作台", Width: g.Width, Height: g.Height, MinWidth: 800, MinHeight: 600, InitialPosition: application.WindowXY, X: g.X, Y: g.Y, URL: windowURL, BackgroundColour: application.NewRGB(242, 243, 237)})
		workbench = w
		w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
			mu.Lock()
			if workbench == w {
				workbench = nil
			}
			mu.Unlock()
		})
		// Keep standard OS titlebar drag, resize and maximize semantics. Only
		// persist a normal (not maximized/minimized) rectangle, debounced.
		var geometryMu sync.Mutex
		var timer *time.Timer
		save := func(_ *application.WindowEvent) {
			if w.IsMaximised() || w.IsMinimised() {
				return
			}
			x, y := w.Position()
			width, height := w.Size()
			g := windowGeometry{x, y, width, height}
			geometryMu.Lock()
			defer geometryMu.Unlock()
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(400*time.Millisecond, func() { _ = a.saveGeometry("workbench", g) })
		}
		w.OnWindowEvent(events.Common.WindowDidMove, save)
		w.OnWindowEvent(events.Common.WindowDidResize, save)
		w.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
			geometryMu.Lock()
			defer geometryMu.Unlock()
			if timer != nil {
				timer.Stop()
			}
			if !w.IsMaximised() && !w.IsMinimised() {
				x, y := w.Position()
				width, height := w.Size()
				_ = a.saveGeometry("workbench", windowGeometry{x, y, width, height})
			}
		})
	}
	open(link)
	placePet := func() {
		if pet == nil {
			return
		}
		if screen := host.Screen.GetPrimary(); screen != nil {
			area := screen.WorkArea
			if g, ok := a.loadGeometry("teacher"); ok {
				w, h := pet.Size()
				pet.SetPosition(max(area.X, min(area.X+max(0, area.Width-w), g.X)), max(area.Y, min(area.Y+max(0, area.Height-h), g.Y)))
				return
			}
			pet.SetPosition(area.X+max(0, area.Width-236), area.Y+max(0, area.Height-448))
			x, y := pet.Position()
			b, _ := json.Marshal(map[string]any{"screen": screen, "x": x, "y": y})
			_ = os.WriteFile(filepath.Join(dir, "teacher-window.json"), b, 0600)
		}
	}
	showPet := func() {
		visible.Store(true)
		if pet == nil {
			screen := host.Screen.GetPrimary()
			x, y := 0, 0
			if screen != nil {
				x = max(0, screen.WorkArea.Width-236)
				y = max(0, screen.WorkArea.Height-448)
			}
			pet = host.Window.NewWithOptions(application.WebviewWindowOptions{InitialPosition: application.WindowXY, Screen: screen, X: x, Y: y, Name: "teacher-companion", Title: "桌面教师 · 原创教师", Width: 220, Height: 440, DisableResize: true, AlwaysOnTop: true, Frameless: true, BackgroundType: application.BackgroundTypeTransparent, Mac: application.MacWindow{Backdrop: application.MacBackdropTransparent}, Windows: application.WindowsWindow{WebView2CompositionHosting: true, DisableFramelessWindowDecorations: true, HiddenOnTaskbar: true}, URL: a.origin + "/pet.html#" + a.token, BackgroundColour: application.NewRGBA(0, 0, 0, 0)})
			pet.OnWindowEvent(events.Common.WindowRuntimeReady, func(e *application.WindowEvent) { placePet() })
			pet.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { visible.Store(false); pet.Hide(); e.Cancel() })
		} else {
			pet.Show()
		}
	}
	showPet()
	host.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) { placePet() })
	desktopState.Lock()
	desktopState.control = func(action, unused string) error {
		mu.Lock()
		defer mu.Unlock()
		switch action {
		case "drag":
			if pet == nil {
				return nativeUnavailable()
			}
			var d struct {
				Phase string `json:"phase"`
				X     int    `json:"x"`
				Y     int    `json:"y"`
			}
			if e := json.Unmarshal([]byte(unused), &d); e != nil {
				return e
			}
			if d.Phase == "begin" {
				dragAnchor.active = true
				dragAnchor.screenX = d.X
				dragAnchor.screenY = d.Y
				dragAnchor.windowX, dragAnchor.windowY = pet.Position()
			}
			if dragAnchor.active && d.Phase != "begin" {
				pet.SetPosition(dragAnchor.windowX+d.X-dragAnchor.screenX, dragAnchor.windowY+d.Y-dragAnchor.screenY)
			}
			if d.Phase == "end" {
				dragAnchor.active = false
				x, y := pet.Position()
				w, h := pet.Size()
				_ = a.saveGeometry("teacher", windowGeometry{X: x, Y: y, Width: w, Height: h})
			}

		case "open":
			// Reuse the existing workbench window (focus current task) instead of
			// spawning a new window/process on every open.
			if workbench != nil {
				workbench.Show()
				workbench.Restore()
				workbench.Focus()
			} else {
				open(a.origin + "/#" + a.token)
			}
		case "open_voice":
			open(a.origin + "/?voice=1#" + a.token)
		case "open_branch":
			open(a.origin + "/?branch=" + unused + "#" + a.token)
		case "show_pet":
			showPet()
		case "size":
			var scale float64
			if json.Unmarshal([]byte(unused), &scale) == nil && pet != nil {
				pet.SetSize(220, int(200+240*scale))
				pet.ExecJS(fmt.Sprintf("document.documentElement.style.setProperty('--teacher-height','%dpx')", int(240*scale)))
			}
		case "hide_pet":
			visible.Store(false)
			if pet != nil {
				x, y := pet.Position()
				w, h := pet.Size()
				_ = a.saveGeometry("teacher", windowGeometry{x, y, w, h})
				pet.Hide()
			}
		case "quit":
			go host.Quit()
		}
		return nil
	}
	desktopState.stop = func() { host.Quit() }
	desktopState.Unlock()
	travelDone := make(chan struct{})
	defer close(travelDone)
	go func() {
		ticker := time.NewTicker(time.Second / 60)
		defer ticker.Stop()
		var motion teacherRoam
		for {
			select {
			case <-travelDone:
				return
			case now := <-ticker.C:
				desktopState.Lock()
				mu.Lock()
				if pet != nil && visible.Load() {
					active := !now.Before(desktopState.dragUntil) && now.Before(desktopState.roamUntil) && resolvedTeacherPosture(desktopState.posture, desktopState.status, desktopState.activityAt, now) == "active"
					screen, e := pet.GetScreen()
					if e != nil || screen == nil {
						screen = host.Screen.GetPrimary()
					}
					if screen != nil {
						if n := host.Screen.GetByID(screen.ID); n != nil {
							screen = n
						}
						r := screen.WorkArea
						x, y := pet.Position()
						w, h := pet.Size()
						speed := desktopState.roamSpeed
						if speed == 0 {
							speed = 120
						}
						nx, ny, d := motion.step(now, x, y, r.X, r.Y, r.X+max(0, r.Width-w), r.Y+max(0, r.Height-h), speed, active)
						if d != "" {
							desktopState.direction = d
						}
						if nx != x || ny != y {
							pet.SetPosition(nx, ny)
						}
					}
				}
				mu.Unlock()
				desktopState.Unlock()
			}
		}
	}()
	defer func() {
		desktopState.Lock()
		desktopState.control = nil
		desktopState.stop = nil
		desktopState.Unlock()
	}()
	return host.Run()
}

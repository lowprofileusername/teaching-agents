//go:build desktop

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

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
	host := application.New(application.Options{
		Name: "教学创新备赛工作台", Description: "报告、课堂实录与教学汇报审阅",
		Mac:     application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
		Windows: application.WindowsOptions{WebviewUserDataPath: cache},
	})
	var mu sync.Mutex
	var pet *application.WebviewWindow
	count := 0
	var visible atomic.Bool
	open := func(windowURL string) {
		count++
		host.Window.NewWithOptions(application.WebviewWindowOptions{Name: fmt.Sprintf("workbench-%d", count), Title: "教学创新备赛工作台", Width: 1440, Height: 860, MinWidth: 980, MinHeight: 680, URL: windowURL, BackgroundColour: application.NewRGB(242, 243, 237)})
	}
	open(link)
	placePet := func() {
		if pet == nil {
			return
		}
		if screen := host.Screen.GetPrimary(); screen != nil {
			area := screen.WorkArea
			pet.SetPosition(area.X+max(0, area.Width-276), area.Y+max(0, area.Height-548))
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
				x = max(0, screen.WorkArea.Width-276)
				y = max(0, screen.WorkArea.Height-548)
			}
			pet = host.Window.NewWithOptions(application.WebviewWindowOptions{InitialPosition: application.WindowXY, Screen: screen, X: x, Y: y, Name: "teacher-companion", Title: "桌面教师 · 原创教师", Width: 260, Height: 540, DisableResize: true, AlwaysOnTop: true, Frameless: true, BackgroundType: application.BackgroundTypeTransparent, Mac: application.MacWindow{Backdrop: application.MacBackdropTransparent}, Windows: application.WindowsWindow{WebView2CompositionHosting: true, DisableFramelessWindowDecorations: true, HiddenOnTaskbar: true}, URL: a.origin + "/pet.html#" + a.token, BackgroundColour: application.NewRGBA(0, 0, 0, 0)})
			pet.OnWindowEvent(events.Common.WindowRuntimeReady, func(e *application.WindowEvent) { placePet() })
			pet.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { visible.Store(false); pet.Hide(); e.Cancel() })
		} else {
			pet.Show()
			placePet()
		}
	}
	showPet()
	host.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(event *application.ApplicationEvent) {
		if screen := host.Screen.GetPrimary(); screen != nil {
			area := screen.WorkArea
			pet.SetPosition(area.X+max(0, area.Width-276), area.Y+max(0, area.Height-548))
		}
	})
	desktopState.Lock()
	desktopState.control = func(action, unused string) error {
		mu.Lock()
		defer mu.Unlock()
		switch action {
		case "open":
			open(link)
		case "open_branch":
			open(a.origin + "/?branch=" + unused + "#" + a.token)
		case "show_pet":
			showPet()
		case "step":
			if visible.Load() && pet != nil {
				x, y := pet.Position()
				screen, e := pet.GetScreen()
				if screen != nil {
					if normalized := host.Screen.GetByID(screen.ID); normalized != nil {
						screen = normalized
					}
				}
				if e != nil || screen == nil {
					screen = host.Screen.GetPrimary()
				}
				if screen != nil {
					area := screen.WorkArea
					dx := 2
					if unused == "left" {
						dx = -2
					}
					x += dx
					lo := area.X
					hi := area.X + max(0, area.Width-260)
					if x <= lo {
						x = lo
						desktopState.direction = "right"
					}
					if x >= hi {
						x = hi
						desktopState.direction = "left"
					}
					pet.SetPosition(x, y)
				}
			}
		case "hide_pet":
			visible.Store(false)
			if pet != nil {
				pet.Hide()
			}
		case "quit":
			go host.Quit()
		}
		return nil
	}
	desktopState.stop = func() { host.Quit() }
	desktopState.Unlock()
	defer func() {
		desktopState.Lock()
		desktopState.control = nil
		desktopState.stop = nil
		desktopState.Unlock()
	}()
	return host.Run()
}

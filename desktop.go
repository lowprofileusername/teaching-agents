package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"
)

// Native shells register this runner; headless builds keep the same HTTP API.
var runDesktop func(*App, string) error
var desktopState struct {
	sync.Mutex
	control    func(string, string) error
	stop       func()
	template   string
	status     string
	posture    string
	activityAt time.Time
	direction  string
	roamUntil  time.Time
	roamSpeed  float64
	roamScale  float64
	dragUntil  time.Time
	branches   map[string]desktopBranch
}

type desktopBranch struct {
	Data json.RawMessage
	At   time.Time
}

func (a *App) desktop(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Action   string  `json:"action"`
		Template string  `json:"template"`
		State    string  `json:"state"`
		Posture  string  `json:"posture"`
		Active   bool    `json:"active"`
		Speed    float64 `json:"speed"`
		Scale    float64 `json:"scale"`
		Phase    string  `json:"phase"`
		X        int     `json:"x"`
		Y        int     `json:"y"`
	}
	if !a.read(w, r, &x) {
		return
	}
	desktopState.Lock()
	defer desktopState.Unlock()
	if desktopState.template == "" {
		desktopState.template = "female"
	}
	if desktopState.status == "" {
		desktopState.status = "idle"
	}
	if desktopState.posture == "" {
		desktopState.posture = "auto"
	}
	if desktopState.activityAt.IsZero() {
		desktopState.activityAt = time.Now()
	}
	if x.Action == "status" {
		jsonOut(w, 200, map[string]any{"native": desktopState.control != nil, "template": desktopState.template, "state": desktopState.status, "posture_mode": desktopState.posture, "direction": desktopState.direction, "posture": resolvedTeacherPosture(desktopState.posture, desktopState.status, desktopState.activityAt, time.Now())})
		return
	}
	if x.Action == "posture" {
		if !oneOf(x.Posture, "auto", "sit", "active") {
			fail(w, 400, "未知教师动作")
			return
		}
		desktopState.posture = x.Posture
		desktopState.activityAt = time.Now()
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if x.Action == "state" {
		if !oneOf(x.State, "idle", "retrieving", "reviewing", "thinking", "answering", "done", "error") {
			fail(w, 400, "未知任务状态")
			return
		}
		if desktopState.status != x.State {
			desktopState.activityAt = time.Now()
		}
		desktopState.status = x.State
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if x.Action == "template" {
		if !oneOf(x.Template, "female", "male") {
			fail(w, 400, "请选择原创女教师或原创男教师")
			return
		}
		desktopState.template = x.Template
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if x.Action == "roam" {
		desktopState.roamUntil = time.Time{}
		if x.Active {
			desktopState.roamUntil = time.Now().Add(750 * time.Millisecond)
		}
		desktopState.roamSpeed = max(40, min(220, x.Speed))
		jsonOut(w, 200, map[string]any{"direction": desktopState.direction})
		return
	}
	if x.Action == "drag" {
		if !oneOf(x.Phase, "begin", "move", "end") || x.X < -100000 || x.X > 100000 || x.Y < -100000 || x.Y > 100000 {
			fail(w, 400, "无效拖动参数")
			return
		}
		if desktopState.control == nil {
			fail(w, 409, "原生窗口未就绪")
			return
		}
		desktopState.roamUntil = time.Time{}
		desktopState.dragUntil = time.Now().Add(2 * time.Second)
		if x.Phase == "end" {
			desktopState.dragUntil = time.Time{}
		}
		raw, _ := json.Marshal(map[string]any{"phase": x.Phase, "x": x.X, "y": x.Y})
		if e := desktopState.control("drag", string(raw)); e != nil {
			fail(w, 500, e.Error())
			return
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if x.Action == "size" {
		desktopState.roamScale = max(.65, min(1.2, x.Scale))
		if desktopState.control != nil {
			raw, _ := json.Marshal(desktopState.roamScale)
			if e := desktopState.control("size", string(raw)); e != nil {
				fail(w, 500, e.Error())
				return
			}
		}
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if x.Action == "step" {
		if desktopState.direction == "" {
			desktopState.direction = "left"
		}
		if desktopState.control != nil && resolvedTeacherPosture(desktopState.posture, desktopState.status, desktopState.activityAt, time.Now()) != "sit" {
			if e := desktopState.control("step", desktopState.direction); e != nil {
				fail(w, 500, e.Error())
				return
			}
		}
		jsonOut(w, 200, map[string]string{"direction": desktopState.direction})
		return
	}
	if desktopState.control == nil {
		fail(w, 409, "此构建为浏览器模式；桌面教师需要原生桌面构建")
		return
	}
	if !oneOf(x.Action, "open", "open_voice", "show_pet", "hide_pet", "quit") {
		fail(w, 400, "未知窗口操作")
		return
	}
	if e := desktopState.control(x.Action, ""); e != nil {
		fail(w, 500, e.Error())
		return
	}
	jsonOut(w, 200, map[string]bool{"ok": true})
}
func nativeUnavailable() error { return errors.New("原生窗口尚未就绪") }

// Appearance is independent from task execution: sitting never cancels a request.
// Automatic rest starts after 90 idle seconds and ends when work resumes.
func resolvedTeacherPosture(mode, status string, activityAt, now time.Time) string {
	if mode == "sit" {
		return "sit"
	}
	if mode == "active" {
		return "active"
	}
	if oneOf(status, "idle", "done", "error") && now.Sub(activityAt) >= 90*time.Second {
		return "sit"
	}
	return "active"
}

package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

// Branch payloads stay in memory, require the process session token and can
// be consumed only once. Neither disk profiles nor API credentials are copied.
func (a *App) windows(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Action   string          `json:"action"`
		ID       string          `json:"id"`
		Snapshot json.RawMessage `json:"snapshot"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&x) != nil {
		fail(w, 400, "窗口上下文无效或超过2MB")
		return
	}
	desktopState.Lock()
	defer desktopState.Unlock()
	now := time.Now()
	if desktopState.branches == nil {
		desktopState.branches = map[string]desktopBranch{}
	}
	for id, b := range desktopState.branches {
		if now.Sub(b.At) > time.Minute {
			delete(desktopState.branches, id)
		}
	}
	if x.Action == "consume" {
		b, ok := desktopState.branches[x.ID]
		if !ok {
			fail(w, 404, "追问上下文已读取或过期，请重新打开追问窗口")
			return
		}
		delete(desktopState.branches, x.ID)
		jsonOut(w, 200, b.Data)
		return
	}
	if x.Action != "open" {
		fail(w, 400, "未知窗口操作")
		return
	}
	id := ""
	if len(x.Snapshot) > 0 && string(x.Snapshot) != "null" {
		var meta struct {
			Version int `json:"version"`
		}
		if json.Unmarshal(x.Snapshot, &meta) != nil || meta.Version != 1 {
			fail(w, 400, "不支持的窗口上下文格式")
			return
		}
		if len(desktopState.branches) >= 8 {
			fail(w, 429, "待打开的追问窗口过多，请稍后重试")
			return
		}
		b := make([]byte, 16)
		if _, e := rand.Read(b); e != nil {
			fail(w, 500, "无法创建追问编号")
			return
		}
		id = hex.EncodeToString(b)
		desktopState.branches[id] = desktopBranch{Data: x.Snapshot, At: now}
	}
	link := a.origin + "/"
	if id != "" {
		link += "?branch=" + id
	}
	link += "#" + a.token
	native := desktopState.control != nil
	if native {
		action := "open"
		if id != "" {
			action = "open_branch"
		}
		if e := desktopState.control(action, id); e != nil {
			delete(desktopState.branches, id)
			fail(w, 500, e.Error())
			return
		}
	}
	jsonOut(w, 200, map[string]any{"native": native, "url": link})
}

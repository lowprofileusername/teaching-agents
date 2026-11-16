package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

func (a *App) moduleTasks(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Action   string          `json:"action"`
		Module   string          `json:"module"`
		ID       string          `json:"id"`
		Title    string          `json:"title"`
		Revision int             `json:"revision"`
		State    json.RawMessage `json:"state"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<20)).Decode(&x) != nil {
		fail(w, 400, "任务数据无效或过大")
		return
	}
	if !oneOf(x.Module, "overview", "report", "lesson", "presentation", "consistency", "replay", "library") {
		fail(w, 400, "未知工作模块")
		return
	}
	db, e := a.workspaceDB()
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	defer db.Close()
	_, e = db.Exec(`CREATE TABLE IF NOT EXISTS module_task(id TEXT PRIMARY KEY,module TEXT NOT NULL,title TEXT NOT NULL,revision INTEGER NOT NULL,updated TEXT NOT NULL,state TEXT NOT NULL); CREATE INDEX IF NOT EXISTS module_task_order ON module_task(module,updated DESC);`)
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	if x.Action == "list" {
		rows, e := db.Query(`SELECT id,title,revision,updated FROM module_task WHERE module=? ORDER BY updated DESC LIMIT 200`, x.Module)
		if e != nil {
			fail(w, 500, e.Error())
			return
		}
		defer rows.Close()
		list := []map[string]any{}
		for rows.Next() {
			var id, title, at string
			var rev int
			if e = rows.Scan(&id, &title, &rev, &at); e != nil {
				fail(w, 500, e.Error())
				return
			}
			list = append(list, map[string]any{"id": id, "title": title, "revision": rev, "updated": at})
		}
		if rows.Err() != nil {
			fail(w, 500, rows.Err().Error())
			return
		}
		jsonOut(w, 200, map[string]any{"tasks": list})
		return
	}
	if !workspaceID.MatchString(x.ID) {
		fail(w, 400, "任务编号无效")
		return
	}
	if x.Action == "load" {
		var raw, title, at string
		var rev int
		e = db.QueryRow(`SELECT state,title,revision,updated FROM module_task WHERE id=? AND module=?`, x.ID, x.Module).Scan(&raw, &title, &rev, &at)
		if e == sql.ErrNoRows {
			fail(w, 404, "任务不存在")
			return
		}
		if e != nil {
			fail(w, 500, e.Error())
			return
		}
		jsonOut(w, 200, map[string]any{"id": x.ID, "title": title, "revision": rev, "updated": at, "state": json.RawMessage(raw)})
		return
	}
	if x.Action != "save" || len([]rune(x.Title)) > 200 {
		fail(w, 400, "未知操作或任务名称过长")
		return
	}
	var state struct {
		Version int                        `json:"version"`
		Session map[string]json.RawMessage `json:"session"`
	}
	if json.Unmarshal(x.State, &state) != nil || state.Version != 1 || state.Session == nil {
		fail(w, 400, "任务快照无效")
		return
	}
	at := time.Now().UTC().Format(time.RFC3339Nano)
	var res sql.Result
	if x.Revision == 0 {
		res, e = db.Exec(`INSERT OR IGNORE INTO module_task VALUES(?,?,?,1,?,?)`, x.ID, x.Module, x.Title, at, string(x.State))
	} else {
		res, e = db.Exec(`UPDATE module_task SET title=?,revision=revision+1,updated=?,state=? WHERE id=? AND module=? AND revision=?`, x.Title, at, string(x.State), x.ID, x.Module, x.Revision)
	}
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		fail(w, 409, "此任务已在另一窗口更新，请刷新列表或另存为新任务，未覆盖记录")
		return
	}
	jsonOut(w, 200, map[string]any{"id": x.ID, "revision": x.Revision + 1, "updated": at})
}

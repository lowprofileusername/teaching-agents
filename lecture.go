package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

func lectureTimeline(ctx context.Context, db *sql.DB, tag string) (map[string]any, error) {
	video, e := rowsJSON(ctx, db, `SELECT tag,title,teacher,institution,course,duration FROM video WHERE tag=?`, tag)
	if e != nil {
		return nil, e
	}
	if len(video) != 1 {
		return nil, fmt.Errorf("讲座不存在")
	}
	entries, e := rowsJSON(ctx, db, `SELECT p.id AS page_id,p.page_no,p.title,o.t0,o.t1 FROM page p JOIN page_occurrence o ON o.page_id=p.id WHERE p.tag=? AND o.t1>o.t0 ORDER BY o.t0,o.t1`, tag)
	if e != nil {
		return nil, e
	}
	images := []map[string]any{}
	if isClaudeLibrary(db) {
		frames, err := rowsJSON(ctx, db, `SELECT f.id AS frame_id,CASE WHEN f.page_no IS NOT NULL THEN f.tag||':P'||f.page_no ELSE '' END AS page_id,f.page_no,coalesce(p.title,f.kind,'非幻灯片画面') AS title,f.t_start AS t0,f.t_end AS t1,'claude:frame-image:'||f.id AS asset_id,'来源库分段代表截图' AS image_kind FROM main.frame f LEFT JOIN temp.page p ON p.tag=f.tag AND p.page_no=f.page_no WHERE f.tag=? AND f.t_end>f.t_start AND length(f.image)>0 ORDER BY f.t_start,f.t_end`, tag)
		if err != nil {
			return nil, err
		}
		// Frame intervals preserve revisits and non-slide scenes. Page occurrences
		// remain a fallback only where the frame timeline has no image.
		for _, f := range frames {
			f["priority"] = 1
		}
		entries = append(entries, frames...)
		images, e = rowsJSON(ctx, db, `SELECT id AS page_id,'claude:page-image:'||source_row_id AS asset_id,t_start AS source_seconds,'来源库去重页代表截图' AS kind FROM temp.page WHERE tag=? AND has_image=1`, tag)
	} else {
		images, e = rowsJSON(ctx, db, `SELECT i.page_id,i.asset_id,i.source_seconds,i.kind FROM page_image i JOIN page p ON p.id=i.page_id WHERE p.tag=? ORDER BY i.source_seconds`, tag)
	}
	if e != nil {
		return nil, e
	}
	utterances, e := rowsJSON(ctx, db, `SELECT id,t0,t1,text,version,quality_status FROM utterance WHERE tag=? AND t1>t0 ORDER BY t0,t1`, tag)
	if e != nil {
		return nil, e
	}
	return map[string]any{"video": video[0], "entries": entries, "images": images, "utterances": utterances, "notice": "依据资料库的画面出现区间同步代表截图与原声；不是原视频逐帧播放。空档与时间重叠会标明，转写未获整场听校。"}, nil
}
func (a *App) libraryLecture(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Tag             string  `json:"tag"`
		ExpectedLibrary *string `json:"expected_library"`
	}
	if !a.read(w, r, &x) {
		return
	}
	if x.Tag == "" {
		fail(w, 400, "请选择一场讲座")
		return
	}
	db, path, e := getLibrary()
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	if x.ExpectedLibrary != nil && *x.ExpectedLibrary != path {
		fail(w, 409, "资料库已改变，请重新选择讲座")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	out, e := lectureTimeline(ctx, db, x.Tag)
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	jsonOut(w, 200, out)
}

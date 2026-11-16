package main

import (
	"context"
	"database/sql"
)

func isWorkBuddyLibrary(ctx context.Context, db *sql.DB) bool {
	var n int
	if db.QueryRowContext(ctx, "SELECT 1 FROM wb_acceptance LIMIT 1").Scan(&n) == nil {
		return true
	}
	var v string
	return db.QueryRowContext(ctx, "SELECT value FROM meta WHERE key='wb_format'").Scan(&v) == nil && v == "workbuddy"
}
func workbuddyReadme(ctx context.Context, db *sql.DB) string {
	var b string
	_ = db.QueryRowContext(ctx, "SELECT body FROM readme WHERE key='workbuddy'").Scan(&b)
	return b
}
func workbuddyQuality(ctx context.Context, db *sql.DB) map[string]any {
	var version string
	_ = db.QueryRowContext(ctx, "SELECT value FROM meta WHERE key='wb_format_version'").Scan(&version)
	checks, _ := rowsJSON(ctx, db, "SELECT check_id,name,status,note FROM wb_acceptance ORDER BY check_id")
	summary, _ := rowsJSON(ctx, db, "SELECT scope,total,flagged,note FROM wb_quality_summary ORDER BY scope")
	return map[string]any{"version": version, "acceptance": checks, "summary": summary, "notice": "以下为资料库作者提供的验收记录，不等于本程序重新验收或真人听校。"}
}

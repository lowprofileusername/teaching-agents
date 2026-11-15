package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
)

// Cache only plain-text scan results in RAM; never cache media or write local files.
// SQLite data_version invalidates entries after another connection commits changes.
var textRowsCache struct {
	sync.Mutex
	db      *sql.DB
	version int64
	rows    map[string][]map[string]any
}

func cachedTextRows(ctx context.Context, db *sql.DB, q string, args ...any) ([]map[string]any, error) {
	textRowsCache.Lock()
	defer textRowsCache.Unlock()
	var version int64
	if err := db.QueryRowContext(ctx, "PRAGMA data_version").Scan(&version); err != nil {
		return nil, err
	}
	if textRowsCache.db != db || textRowsCache.version != version {
		textRowsCache.db = db
		textRowsCache.version = version
		textRowsCache.rows = map[string][]map[string]any{}
	}
	keyBytes, _ := json.Marshal(args)
	key := q + string(keyBytes)
	rows, ok := textRowsCache.rows[key]
	if !ok {
		var err error
		rows, err = rowsJSON(ctx, db, q, args...)
		if err != nil {
			return nil, err
		}
		if len(textRowsCache.rows) >= 16 {
			textRowsCache.rows = map[string][]map[string]any{}
		}
		textRowsCache.rows[key] = rows
	}
	result := make([]map[string]any, len(rows))
	for i, row := range rows {
		copyRow := make(map[string]any, len(row))
		for k, v := range row {
			copyRow[k] = v
		}
		result[i] = copyRow
	}
	return result, nil
}

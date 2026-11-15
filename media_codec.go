package main

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

// Text search never calls this function. Only a requested media object is decoded.
func decodeMedia(ctx context.Context, db *sql.DB, table, id string, b []byte) ([]byte, error) {
	var codec, sha string
	var size int64
	err := db.QueryRowContext(ctx, "SELECT codec,raw_sha256,raw_bytes FROM media_encoding WHERE table_name=? AND record_id=?", table, id).Scan(&codec, &sha, &size)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if err == sql.ErrNoRows || strings.Contains(err.Error(), "no such table") {
			return b, nil
		}
		return nil, err
	} // old DBs have no codec table
	if codec != "zlib" || size < 0 || size > 512<<20 {
		return nil, errors.New("不支持的媒体编码或大小")
	}
	r, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	raw, err := io.ReadAll(io.LimitReader(r, size+1))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	if int64(len(raw)) != size || hex.EncodeToString(sum[:]) != sha {
		return nil, errors.New("无损媒体校验失败")
	}
	return raw, nil
}

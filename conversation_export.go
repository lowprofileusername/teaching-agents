package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A visible, user-selected output directory; never the browser's default SSD
// Downloads folder. No overwrite and no database or API-key export.
func (a *App) conversationExport(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Action    string `json:"action"`
		Directory string `json:"directory"`
		Format    string `json:"format"`
		Content   string `json:"content"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&x) != nil {
		fail(w, 400, "导出内容无效或超过8MB")
		return
	}
	if x.Action == "directory" {
		_, path, err := getLibrary()
		if err != nil {
			if a.profiles == nil {
				fail(w, 409, "请选择导出目录")
				return
			}
			path = a.profiles.path
		}
		jsonOut(w, 200, map[string]string{"directory": filepath.Join(filepath.Dir(path), "对话导出")})
		return
	}
	if x.Action != "save" || !oneOf(x.Format, "md", "txt", "json") || strings.TrimSpace(x.Content) == "" {
		fail(w, 400, "请选择有效的导出格式和对话内容")
		return
	}
	if !filepath.IsAbs(x.Directory) {
		fail(w, 400, "请填写完整的导出文件夹路径")
		return
	}
	dir := filepath.Clean(x.Directory)
	// Its parent must exist, avoiding accidental creation of an unmounted share.
	if info, err := os.Stat(dir); err != nil {
		parent, pe := os.Stat(filepath.Dir(dir))
		if pe != nil || !parent.IsDir() {
			fail(w, 400, "导出目录的父文件夹不存在，请确认共享盘已连接")
			return
		}
		if err = os.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
			fail(w, 400, "无法创建导出文件夹，请检查写入权限")
			return
		}
	} else if !info.IsDir() {
		fail(w, 400, "导出路径必须是文件夹")
		return
	}
	nonce := make([]byte, 4)
	if _, err := rand.Read(nonce); err != nil {
		fail(w, 500, "无法生成文件编号")
		return
	}
	name := fmt.Sprintf("教学资料助手_对话_%s_%s.%s", time.Now().Format("20060102_150405"), hex.EncodeToString(nonce), x.Format)
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fail(w, 400, "无法写入导出目录")
		return
	}
	_, err = f.WriteString(x.Content)
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(path)
		fail(w, 500, "写入导出文件失败")
		return
	}
	jsonOut(w, 200, map[string]any{"path": path, "bytes": len(x.Content), "format": x.Format})
}

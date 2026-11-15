package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

const keyService = "teaching-agents.api.v1"

var secretGet = keyring.Get
var secretSet = keyring.Set
var secretDelete = keyring.Delete

type APIProfile struct {
	ModelOptions
	ID     string `json:"id"`
	Name   string `json:"name"`
	Base   string `json:"base"`
	Model  string `json:"model"`
	KeyRef string `json:"keyRef,omitempty"`
}
type ProfileFile struct {
	LibraryPath string       `json:"libraryPath,omitempty"`
	Active      string       `json:"active"`
	Profiles    []APIProfile `json:"profiles"`
}
type ProfileStore struct {
	mu   sync.Mutex
	path string
}

func (s *ProfileStore) read() (ProfileFile, error) {
	var p ProfileFile
	b, e := os.ReadFile(s.path)
	if os.IsNotExist(e) {
		return p, nil
	}
	if e != nil {
		return p, e
	}
	e = json.Unmarshal(b, &p)
	return p, e
}
func (s *ProfileStore) write(p ProfileFile) error {
	b, e := json.MarshalIndent(p, "", "  ")
	if e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(s.path), ".profiles-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(name, s.path)
}
func newID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func (a *App) initProfiles() {
	path := os.Getenv("TEACHING_AGENT_PROFILES")
	if path == "" {
		exe, e := os.Executable()
		if e != nil {
			return
		}
		dir := filepath.Dir(exe)
		if runtime.GOOS == "darwin" && strings.Contains(exe, ".app/Contents/MacOS/") {
			dir = filepath.Dir(filepath.Dir(filepath.Dir(dir)))
		}
		path = filepath.Join(dir, "teaching-agent-profiles.json")
	}
	a.profiles = &ProfileStore{path: path}
	p, e := a.profiles.read()
	if e != nil {
		return
	}
	for _, v := range p.Profiles {
		if v.ID == p.Active {
			key := ""
			if v.KeyRef != "" {
				key, e = secretGet(keyService, v.KeyRef)
				if e != nil {
					return
				}
			}
			a.cfg.ModelOptions = v.ModelOptions
			a.cfg.Base = v.Base
			a.cfg.Model = v.Model
			a.cfg.Key = key
		}
	}
}
func publicProfile(p APIProfile) map[string]any {
	return map[string]any{"id": p.ID, "name": p.Name, "base": p.Base, "model": p.Model, "hasKey": p.KeyRef != "", "thinkingMode": p.ThinkingMode, "thinkingFormat": p.ThinkingFormat, "reasoningEffort": p.ReasoningEffort, "contextWindow": p.ContextWindow, "maxTokens": p.MaxTokens}
}
func (a *App) profileAPI(w http.ResponseWriter, r *http.Request) {
	var x struct {
		ModelOptions
		Action   string `json:"action"`
		ID       string `json:"id"`
		Name     string `json:"name"`
		Base     string `json:"base"`
		Model    string `json:"model"`
		Key      string `json:"key"`
		ClearKey bool   `json:"clearKey"`
	}
	if !a.read(w, r, &x) {
		return
	}
	s := a.profiles
	if s == nil {
		fail(w, 503, "配置存储尚未初始化")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Multiple persona executables may share this directory. Never overwrite concurrently.
	lock, e := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		fail(w, 409, "配置目录不可写或另一个窗口正在保存，请稍后重试")
		return
	}
	lock.Close()
	defer os.Remove(s.path + ".lock")
	p, e := s.read()
	if e != nil {
		fail(w, 500, "无法读取配置文件；原文件未覆盖")
		return
	}
	idx := -1
	for i, v := range p.Profiles {
		if v.ID == x.ID {
			idx = i
			break
		}
	}
	var selected *APIProfile
	switch x.Action {
	case "", "list":
	case "save":
		base, err := endpoint(x.Base)
		if err != nil {
			fail(w, 400, err.Error())
			return
		}
		x.Name = strings.TrimSpace(x.Name)
		x.Model = strings.TrimSpace(x.Model)
		if x.Name == "" || x.Model == "" || len(x.Name) > 200 || len(x.Model) > 500 || len(x.Key) > 2000 {
			fail(w, 400, "请填写配置名称、模型；密钥不超过2000字节")
			return
		}
		if x.ID != "" && idx < 0 {
			fail(w, 404, "配置不存在，请新建")
			return
		}
		if err := x.ModelOptions.normalize(); err != nil {
			fail(w, 400, err.Error())
			return
		}
		v := APIProfile{ModelOptions: x.ModelOptions, ID: newID(), Name: x.Name, Base: base, Model: x.Model}
		oldRef := ""
		if idx >= 0 {
			v.ID = p.Profiles[idx].ID
			oldRef = p.Profiles[idx].KeyRef
			if base != p.Profiles[idx].Base && oldRef != "" && x.Key == "" && !x.ClearKey {
				fail(w, 400, "API 地址已改变，请重新输入对应密钥，或勾选无密钥")
				return
			}
			v.KeyRef = oldRef
		}
		key := strings.TrimSpace(x.Key)
		if x.ClearKey {
			key = ""
			v.KeyRef = ""
		} else if key != "" {
			v.KeyRef = newID()
			if secretSet(keyService, v.KeyRef, key) != nil {
				fail(w, 500, "系统凭据库保存失败；未将密钥写入明文文件")
				return
			}
		} else if v.KeyRef != "" {
			key, e = secretGet(keyService, v.KeyRef)
			if e != nil {
				fail(w, 409, "本机没有此配置的密钥，请重新输入")
				return
			}
		}
		if idx >= 0 {
			p.Profiles[idx] = v
		} else {
			p.Profiles = append(p.Profiles, v)
		}
		p.Active = v.ID
		if e = s.write(p); e != nil {
			if v.KeyRef != "" && v.KeyRef != oldRef {
				_ = secretDelete(keyService, v.KeyRef)
			}
			fail(w, 500, "保存配置失败，请检查所在目录可写")
			return
		}
		if oldRef != "" && oldRef != v.KeyRef {
			_ = secretDelete(keyService, oldRef)
		}
		a.mu.Lock()
		a.cfg.ModelOptions = v.ModelOptions
		a.cfg.Base = v.Base
		a.cfg.Model = v.Model
		a.cfg.Key = key
		a.mu.Unlock()
		selected = &v
	case "activate":
		if idx < 0 {
			fail(w, 404, "请选择已保存的配置")
			return
		}
		v := p.Profiles[idx]
		key := ""
		if v.KeyRef != "" {
			key, e = secretGet(keyService, v.KeyRef)
			if e != nil {
				fail(w, 409, "此密钥属于其他电脑或无法读取，请在本机重新输入并保存")
				return
			}
		}
		p.Active = v.ID
		if s.write(p) != nil {
			fail(w, 500, "无法保存当前选择")
			return
		}
		a.mu.Lock()
		a.cfg.ModelOptions = v.ModelOptions
		a.cfg.Base = v.Base
		a.cfg.Model = v.Model
		a.cfg.Key = key
		a.mu.Unlock()
		selected = &v
	case "delete":
		if idx < 0 {
			fail(w, 404, "配置不存在")
			return
		}
		v := p.Profiles[idx]
		p.Profiles = append(p.Profiles[:idx], p.Profiles[idx+1:]...)
		active := p.Active == v.ID
		if active {
			p.Active = ""
		}
		if s.write(p) != nil {
			fail(w, 500, "删除配置失败")
			return
		}
		if v.KeyRef != "" {
			_ = secretDelete(keyService, v.KeyRef)
		}
		if active {
			a.mu.Lock()
			a.cfg.Base = ""
			a.cfg.Key = ""
			a.cfg.Model = ""
			a.mu.Unlock()
		}
	default:
		fail(w, 400, "未知操作")
		return
	}
	rows := []map[string]any{}
	for _, v := range p.Profiles {
		rows = append(rows, publicProfile(v))
		if selected == nil && v.ID == p.Active {
			vv := v
			selected = &vv
		}
	}
	a.mu.RLock()
	ready := a.cfg.Base != "" && a.cfg.Model != ""
	a.mu.RUnlock()
	out := map[string]any{"profiles": rows, "active": p.Active, "chatReady": ready, "storage": s.path}
	if selected != nil {
		out["selected"] = publicProfile(*selected)
	}
	jsonOut(w, 200, out)
}

// Test credentials without generating paid text or exposing upstream response bodies.
func (a *App) testAPI(w http.ResponseWriter, r *http.Request) {
	var x struct{}
	if !a.read(w, r, &x) {
		return
	}
	a.mu.RLock()
	c := a.cfg
	a.mu.RUnlock()
	if c.Base == "" {
		fail(w, 409, "请先保存或切换 API 配置")
		return
	}
	req, e := http.NewRequestWithContext(r.Context(), "GET", c.Base+"/models", nil)
	if e != nil {
		fail(w, 400, "无效地址")
		return
	}
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}
	resp, e := a.client.Do(req)
	if e != nil {
		fail(w, 502, "模型接口无法连接")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fail(w, 502, "模型列表返回 "+resp.Status+"；请检查 Base URL 和密钥")
		return
	}
	var d struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, resp.Body, 2<<20)).Decode(&d) != nil {
		fail(w, 502, "接口未返回兼容的模型列表")
		return
	}
	ids := []string{}
	found := false
	for _, m := range d.Data {
		ids = append(ids, m.ID)
		if m.ID == c.Model {
			found = true
		}
	}
	jsonOut(w, 200, map[string]any{"ok": true, "models": ids, "modelFound": found})
}
func (a *App) voiceConfig(w http.ResponseWriter, r *http.Request) {
	var c Config
	if !a.read(w, r, &c) {
		return
	}
	a.mu.Lock()
	a.cfg.VoiceKey = c.VoiceKey
	a.cfg.VoiceID = c.VoiceID
	a.cfg.VoiceAuthorized = c.VoiceAuthorized
	a.cfg.VoiceSample = c.VoiceSample
	a.mu.Unlock()
	jsonOut(w, 200, map[string]bool{"ok": true})
}

func (a *App) initialLibrary() {
	if custom := os.Getenv("TEACHING_LIBRARY_DB"); custom != "" {
		_ = openLibrary(custom)
		return
	}
	if a.profiles != nil {
		if p, e := a.profiles.read(); e == nil && p.LibraryPath != "" {
			_ = openLibrary(p.LibraryPath)
			return
		}
	}
	autoLibrary()
}
func (a *App) saveLibraryPath(path string) bool {
	if a.profiles == nil {
		return false
	}
	s := a.profiles
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, e := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return false
	}
	lock.Close()
	defer os.Remove(s.path + ".lock")
	p, e := s.read()
	if e != nil {
		return false
	}
	p.LibraryPath = path
	return s.write(p) == nil
}

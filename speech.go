package main

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed speech/qwen_worker.py
var qwenWorker []byte

type voicePack struct {
	Version      int    `json:"version"`
	Backend      string `json:"backend"`
	Python       string `json:"python,omitempty"`
	Engine       string `json:"engine,omitempty"`
	Model        string `json:"model"`
	SitePackages string `json:"site_packages,omitempty"`
}
type localSpeech struct {
	gate    sync.Mutex // serializes generation; TryLock rejects overlapping windows
	mu      sync.Mutex
	cmd     *exec.Cmd
	input   io.WriteCloser
	output  *bufio.Scanner
	logfile *os.File
	idle    *time.Timer
}

func (a *App) voiceSettingsPath() string {
	if a.profiles == nil {
		return ""
	}
	return filepath.Join(filepath.Dir(a.profiles.path), "local-voice.json")
}
func (a *App) voiceDirectory() string {
	if d := os.Getenv("TEACHING_AGENT_VOICE_PACK"); d != "" {
		return d
	}
	if b, e := os.ReadFile(a.voiceSettingsPath()); e == nil {
		var s voiceSettings
		if json.Unmarshal(b, &s) == nil && s.Directory != "" {
			return s.Directory
		}
	}
	if a.profiles != nil {
		return filepath.Join(filepath.Dir(a.profiles.path), "qwen-tts")
	}
	return ""
}
func packFile(root, name string) (string, error) {
	// Manifest paths are portable: reject both Windows and POSIX roots on every host.
	name = strings.ReplaceAll(name, `\`, "/")
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, ":") || filepath.IsAbs(name) {
		return "", errors.New("语音包路径无效")
	}
	clean := filepath.Clean(filepath.FromSlash(name))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("语音包路径越界")
	}
	return filepath.Join(root, clean), nil
}
func readVoicePack(dir string) (voicePack, error) {
	var p voicePack
	if !filepath.IsAbs(dir) {
		return p, errors.New("请选择语音包的完整路径")
	}
	b, e := os.ReadFile(filepath.Join(dir, "voice-pack.json"))
	if e != nil {
		return p, errors.New("尚未安装 Qwen3-TTS 语音包，请选择包含 voice-pack.json 的目录")
	}
	if len(b) > 16384 || json.Unmarshal(b, &p) != nil || p.Version != 1 || !oneOf(p.Backend, "mlx", "torch") || (p.Python == "") == (p.Engine == "") {
		return p, errors.New("语音包清单不兼容")
	}
	if p.SitePackages != "" {
		site, e := packFile(dir, p.SitePackages)
		if e != nil {
			return p, e
		}
		if info, e := os.Stat(site); e != nil || !info.IsDir() {
			return p, errors.New("语音包缺少运行依赖")
		}
	}
	model, e := packFile(dir, p.Model)
	if e != nil {
		return p, e
	}
	if _, e = os.Stat(filepath.Join(model, "config.json")); e != nil {
		return p, errors.New("语音包缺少模型配置")
	}
	name := p.Engine
	if name == "" {
		name = p.Python
	}
	bin, e := packFile(dir, name)
	if e != nil {
		return p, e
	}
	if info, e := os.Stat(bin); e != nil || info.IsDir() {
		return p, errors.New("语音包缺少本地运行引擎")
	}
	return p, nil
}
func (s *localSpeech) closeLocked() {
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	if s.cmd != nil {
		if s.input != nil {
			_ = s.input.Close()
		}
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
	if s.logfile != nil {
		_ = s.logfile.Close()
	}
	s.cmd = nil
	s.input = nil
	s.output = nil
	s.logfile = nil
}
func (s *localSpeech) close() { s.mu.Lock(); defer s.mu.Unlock(); s.closeLocked() }
func (s *localSpeech) line(ctx context.Context) (map[string]any, error) {
	type answer struct {
		x map[string]any
		e error
	}
	ch := make(chan answer, 1)
	scanner := s.output
	go func() {
		if !scanner.Scan() {
			ch <- answer{e: errors.New("本地语音进程已退出，请查看语音包中的 runtime-cache/worker.log")}
			return
		}
		var x map[string]any
		e := json.Unmarshal(scanner.Bytes(), &x)
		if e == nil {
			if msg, ok := x["error"].(string); ok {
				e = errors.New(msg)
			}
		}
		ch <- answer{x, e}
	}()
	select {
	case out := <-ch:
		return out.x, out.e
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (s *localSpeech) start(ctx context.Context, dir string, p voicePack) error {
	if s.cmd != nil {
		return nil
	}
	cache := filepath.Join(dir, "runtime-cache")
	if e := os.MkdirAll(cache, 0700); e != nil {
		return e
	}
	for _, n := range []string{"tmp", "hf", "torch", "numba", "mpl"} {
		if e := os.MkdirAll(filepath.Join(cache, n), 0700); e != nil {
			return e
		}
	}
	worker := filepath.Join(cache, "qwen_worker.py")
	if e := os.WriteFile(worker, qwenWorker, 0600); e != nil {
		return e
	}
	model, _ := packFile(dir, p.Model)
	args := []string{"--model", model, "--backend", p.Backend}
	bin, _ := packFile(dir, p.Engine)
	if p.Python != "" {
		bin, _ = packFile(dir, p.Python)
		args = append([]string{"-B", worker}, args...)
	}
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PYTHONNOUSERSITE=1", "PYTHONDONTWRITEBYTECODE=1", "HF_HUB_OFFLINE=1", "TRANSFORMERS_OFFLINE=1", "HF_HOME="+filepath.Join(cache, "hf"), "XDG_CACHE_HOME="+cache, "TMPDIR="+filepath.Join(cache, "tmp"), "TMP="+filepath.Join(cache, "tmp"), "TEMP="+filepath.Join(cache, "tmp"), "TORCH_HOME="+filepath.Join(cache, "torch"), "NUMBA_CACHE_DIR="+filepath.Join(cache, "numba"), "MPLCONFIGDIR="+filepath.Join(cache, "mpl"), "TOKENIZERS_PARALLELISM=false")
	if p.SitePackages != "" {
		site, _ := packFile(dir, p.SitePackages)
		cmd.Env = append(cmd.Env, "PYTHONPATH="+site)
	}
	logfile, e := os.OpenFile(filepath.Join(cache, "worker.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil {
		return e
	}
	cmd.Stderr = logfile
	input, e := cmd.StdinPipe()
	if e != nil {
		logfile.Close()
		return e
	}
	output, e := cmd.StdoutPipe()
	if e != nil {
		input.Close()
		logfile.Close()
		return e
	}
	if e = cmd.Start(); e != nil {
		input.Close()
		output.Close()
		logfile.Close()
		return e
	}
	s.cmd = cmd
	s.input = input
	s.logfile = logfile
	s.output = bufio.NewScanner(output)
	s.output.Buffer(make([]byte, 65536), 12<<20)
	response, e := s.line(ctx)
	if e != nil {
		return e
	}
	if response["ready"] != true {
		return errors.New("语音引擎未完成初始化")
	}
	return nil
}
func (a *App) localVoice(w http.ResponseWriter, r *http.Request) {
	var x voiceRequest
	if !a.read(w, r, &x) {
		return
	}
	s := &a.speech
	if x.Action == "status" {
		a.voiceStatus(w)
		return
	}

	if !s.gate.TryLock() {
		fail(w, 409, "另一个窗口正在生成语音，请先停止朗读或稍后重试")
		return
	}
	defer s.gate.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	if x.Action == "unload" {
		s.closeLocked()
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}
	if x.Action == "configure" {
		if e := a.saveVoiceSettings(x); e != nil {
			fail(w, 400, e.Error())
			return
		}
		s.closeLocked()
		jsonOut(w, 200, map[string]bool{"ok": true})
		return
	}

	if x.Action != "synthesize" {
		fail(w, 400, "未知语音操作")
		return
	}
	text := strings.TrimSpace(x.Text)
	if len([]rune(text)) == 0 || len([]rune(text)) > 240 {
		fail(w, 400, "每段语音须为1至240字，请分段朗读")
		return
	}
	if !oneOf(x.Template, "female", "male") {
		fail(w, 400, "请选择女教师或男教师")
		return
	}
	settings := a.readVoiceSettings()
	provider := settings.Providers[settings.Engine]
	speaker := voiceFor(provider, x.Template)
	if settings.Engine != "qwen" {
		a.remoteSpeech(w, r, settings.Engine, provider, text, speaker)
		return
	}
	if !oneOf(speaker, qwenSpeakers...) {
		fail(w, 400, "Qwen音色配置无效")
		return
	}

	dir := a.voiceDirectory()
	p, e := readVoicePack(dir)
	if e != nil {
		fail(w, 409, e.Error())
		return
	}
	if s.idle != nil {
		s.idle.Stop()
		s.idle = nil
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	if e = s.start(ctx, dir, p); e != nil {
		s.closeLocked()
		fail(w, 503, e.Error())
		return
	}
	b, _ := json.Marshal(map[string]string{"text": text, "speaker": speaker})
	_, e = s.input.Write(append(b, '\n'))
	var response map[string]any
	if e == nil {
		response, e = s.line(ctx)
	}
	if e != nil {
		s.closeLocked()
		fail(w, 503, e.Error())
		return
	}
	encoded, _ := response["audio"].(string)
	audio, e := base64.StdEncoding.DecodeString(encoded)
	if e != nil || len(audio) < 44 || len(audio) > 8<<20 || string(audio[:4]) != "RIFF" {
		s.closeLocked()
		fail(w, 503, "语音引擎返回无效音频")
		return
	}
	s.idle = time.AfterFunc(5*time.Minute, func() {
		if s.gate.TryLock() {
			defer s.gate.Unlock()
			s.close()
		}
	})
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("X-Voice", speaker)
	w.Header().Set("X-Voice-Engine", "qwen")
	w.Header().Set("X-Generation-Seconds", fmt.Sprint(response["seconds"]))
	w.Write(audio)
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var qwenSpeakers = []string{"Serena", "Uncle_Fu", "Vivian", "Dylan", "Eric", "Ryan", "Aiden", "Ono_Anna", "Sohee"}

type voiceProvider struct {
	Base   string `json:"base"`
	Model  string `json:"model"`
	Female string `json:"female"`
	Male   string `json:"male"`
	KeyRef string `json:"key_ref,omitempty"`
	HasKey bool   `json:"has_key,omitempty"`
}
type voiceSettings struct {
	Directory string                   `json:"directory"`
	Engine    string                   `json:"engine"`
	Providers map[string]voiceProvider `json:"providers"`
}
type voiceRequest struct {
	Action    string        `json:"action"`
	Directory string        `json:"directory"`
	Text      string        `json:"text"`
	Template  string        `json:"template"`
	Engine    string        `json:"engine"`
	Provider  voiceProvider `json:"provider"`
	Key       string        `json:"key"`
	ClearKey  bool          `json:"clear_key"`
}

func (a *App) readVoiceSettings() voiceSettings {
	s := voiceSettings{}
	if b, e := os.ReadFile(a.voiceSettingsPath()); e == nil {
		_ = json.Unmarshal(b, &s)
	}
	if s.Engine == "" {
		s.Engine = "compatible"
	}
	if s.Providers == nil {
		s.Providers = map[string]voiceProvider{}
	}
	defaults := map[string]voiceProvider{"qwen": {Female: "Serena", Male: "Uncle_Fu"}, "compatible": {Female: "", Male: ""}, "azure": {Female: "zh-CN-XiaoxiaoNeural", Male: "zh-CN-YunxiNeural"}}
	for k, v := range defaults {
		if _, ok := s.Providers[k]; !ok {
			s.Providers[k] = v
		}
	}
	return s
}
func (a *App) saveVoiceSettings(x voiceRequest) error {
	if !oneOf(x.Engine, "qwen", "compatible", "azure") {
		return errors.New("请选择有效的语音引擎")
	}
	s := a.readVoiceSettings()
	p := x.Provider
	p.HasKey = false
	if x.Engine == "qwen" {
		if !oneOf(p.Female, qwenSpeakers...) || !oneOf(p.Male, qwenSpeakers...) {
			return errors.New("请选择 Qwen3-TTS 预设音色")
		}
		if x.Directory != "" {
			if _, e := readVoicePack(x.Directory); e != nil {
				return e
			}
			s.Directory = x.Directory
		}
		p.Base = ""
		p.Model = ""
		p.KeyRef = ""
	} else {
		base, e := endpoint(p.Base)
		if e != nil {
			return e
		}
		p.Base = base
		if p.Female == "" || p.Male == "" || len(p.Female) > 160 || len(p.Male) > 160 {
			return errors.New("请填写男女教师使用的音色 ID")
		}
		if x.Engine == "compatible" && (p.Model == "" || len(p.Model) > 160) {
			return errors.New("请填写语音模型名称")
		}
		old := s.Providers[x.Engine]
		p.KeyRef = old.KeyRef
		if x.Key == "" && !x.ClearKey && old.Base != "" && old.Base != p.Base && p.KeyRef != "" {
			return errors.New("语音服务地址已改变，请重新输入对应密钥，或选择无需密钥")
		}
		if x.ClearKey {
			p.KeyRef = ""
		} else if x.Key != "" {
			ref := "voice-" + newID()
			if e := secretSet(keyService, ref, x.Key); e != nil {
				return errors.New("无法将语音密钥保存到系统凭据库")
			}
			p.KeyRef = ref
		}
	}
	s.Engine = x.Engine
	s.Providers[x.Engine] = p
	path := a.voiceSettingsPath()
	if path == "" {
		return errors.New("未找到配置目录")
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	f, e := os.CreateTemp(filepath.Dir(path), ".voice-*")
	if e != nil {
		return e
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	_ = f.Chmod(0600)
	_, e = f.Write(b)
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	return os.Rename(tmp, path)
}
func (a *App) voiceStatus(w http.ResponseWriter) {
	s := a.readVoiceSettings()
	dir := a.voiceDirectory()
	pack, e := readVoicePack(dir)
	msg := ""
	if e != nil {
		msg = e.Error()
	}
	for k, p := range s.Providers {
		if p.KeyRef != "" {
			key, err := secretGet(keyService, p.KeyRef)
			p.HasKey = err == nil && key != ""
		}
		p.KeyRef = ""
		s.Providers[k] = p
	}
	jsonOut(w, 200, map[string]any{"installed": e == nil, "directory": dir, "backend": pack.Backend, "error": msg, "engine": s.Engine, "providers": s.Providers, "speakers": qwenSpeakers})
}
func voiceFor(p voiceProvider, template string) string {
	if template == "male" {
		return p.Male
	}
	return p.Female
}
func (a *App) remoteSpeech(w http.ResponseWriter, r *http.Request, engine string, p voiceProvider, text, voice string) {
	base, e := endpoint(p.Base)
	if e != nil {
		fail(w, 409, "请先配置语音引擎地址")
		return
	}
	key := ""
	if p.KeyRef != "" {
		key, e = secretGet(keyService, p.KeyRef)
		if e != nil {
			fail(w, 409, "无法读取语音密钥，请重新保存")
			return
		}
	}
	api := base
	contentType := "application/json"
	var body []byte
	if engine == "compatible" {
		if !strings.HasSuffix(api, "/audio/speech") {
			api += "/audio/speech"
		}
		body, _ = json.Marshal(map[string]string{"model": p.Model, "voice": voice, "input": text, "response_format": "wav"})
	} else {
		if !strings.HasSuffix(api, "/cognitiveservices/v1") {
			api += "/cognitiveservices/v1"
		}
		var v, t bytes.Buffer
		_ = xml.EscapeText(&v, []byte(voice))
		_ = xml.EscapeText(&t, []byte(text))
		body = []byte(`<speak version="1.0" xmlns="http://www.w3.org/2001/10/synthesis" xml:lang="zh-CN"><voice name="` + v.String() + `">` + t.String() + `</voice></speak>`)
		contentType = "application/ssml+xml"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, "POST", api, bytes.NewReader(body))
	if e != nil {
		fail(w, 400, "语音地址无效")
		return
	}
	req.Header.Set("Content-Type", contentType)
	if engine == "azure" {
		req.Header.Set("Ocp-Apim-Subscription-Key", key)
		req.Header.Set("X-Microsoft-OutputFormat", "riff-24khz-16bit-mono-pcm")
		req.Header.Set("User-Agent", "TeachingAgents")
	} else if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	started := time.Now()
	response, e := a.client.Do(req)
	if e != nil {
		fail(w, 502, "无法连接语音服务，请检查语音地址、网络或取消状态")
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		fail(w, 502, fmt.Sprintf("语音服务返回 HTTP %d；请检查语音模型、音色和密钥", response.StatusCode))
		return
	}
	audio, e := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if e != nil || len(audio) < 44 || len(audio) > 8<<20 || string(audio[:4]) != "RIFF" {
		fail(w, 502, "语音服务未返回支持的 WAV 音频，请确认兼容 /audio/speech 与 response_format=wav")
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("X-Voice", voice)
	w.Header().Set("X-Voice-Engine", engine)
	w.Header().Set("X-Generation-Seconds", fmt.Sprintf("%.3f", time.Since(started).Seconds()))
	_, _ = w.Write(audio)
}

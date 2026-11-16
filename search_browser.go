package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coder/websocket"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func searchBrowserExecutable() string {
	paths := []string{}
	if runtime.GOOS == "darwin" {
		paths = []string{"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome", "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"}
	} else if runtime.GOOS == "windows" {
		for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")} {
			paths = append(paths, filepath.Join(base, "Microsoft", "Edge", "Application", "msedge.exe"), filepath.Join(base, "Google", "Chrome", "Application", "chrome.exe"))
		}
	}
	for _, p := range paths {
		if st, e := os.Stat(p); e == nil && !st.IsDir() {
			return p
		}
	}
	for _, name := range []string{"google-chrome", "chromium", "chromium-browser", "msedge"} {
		if p, e := exec.LookPath(name); e == nil {
			return p
		}
	}
	return ""
}

type searchCDP struct {
	conn *websocket.Conn
	id   int
}

func (c *searchCDP) call(ctx context.Context, method string, params any, session string) (json.RawMessage, error) {
	c.id++
	id := c.id
	payload := map[string]any{"id": id, "method": method, "params": params}
	if session != "" {
		payload["sessionId"] = session
	}
	b, _ := json.Marshal(payload)
	if e := c.conn.Write(ctx, websocket.MessageText, b); e != nil {
		return nil, e
	}
	for {
		_, b, e := c.conn.Read(ctx)
		if e != nil {
			return nil, e
		}
		var out struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(b, &out) != nil || out.ID != id {
			continue
		}
		if out.Error != nil {
			return nil, fmt.Errorf("浏览器命令失败：%s", out.Error.Message)
		}
		return out.Result, nil
	}
}
func (a *App) browserSearch(ctx context.Context, engine searchEngine, q string, human bool) ([]webResult, error) {
	exe := searchBrowserExecutable()
	if exe == "" {
		return nil, fmt.Errorf("浏览器保底需要已安装 Chrome 或 Edge")
	}
	parent := filepath.Dir(a.profiles.path)
	if st, e := os.Stat(parent); e != nil || !st.IsDir() {
		return nil, fmt.Errorf("搜索临时目录不可用")
	}
	dir, e := os.MkdirTemp(parent, "search-browser-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(dir)
	args := []string{"--user-data-dir=" + dir, "--remote-debugging-address=127.0.0.1", "--remote-debugging-port=0", "--no-first-run", "--no-default-browser-check", "--disable-background-networking", "--disable-sync", "--disable-crash-reporter", "--disk-cache-dir=" + filepath.Join(dir, "cache"), "--start-minimized", "about:blank"}
	if !human {
		args = append(args, "--headless=new")
	}
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Env = append(os.Environ(), "TMPDIR="+dir, "TEMP="+dir, "TMP="+dir)
	if e = cmd.Start(); e != nil {
		return nil, fmt.Errorf("无法启动隔离搜索浏览器")
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	var endpoint string
	for i := 0; i < 70; i++ {
		b, e := os.ReadFile(filepath.Join(dir, "DevToolsActivePort"))
		if e == nil {
			lines := strings.Split(strings.TrimSpace(string(b)), "\n")
			if len(lines) >= 2 {
				endpoint = "ws://127.0.0.1:" + lines[0] + lines[1]
				break
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	if endpoint == "" {
		return nil, fmt.Errorf("隔离浏览器启动超时")
	}
	conn, _, e := websocket.Dial(ctx, endpoint, &websocket.DialOptions{HTTPClient: &http.Client{Timeout: 5 * time.Second}})
	if e != nil {
		return nil, fmt.Errorf("搜索浏览器连接失败")
	}
	defer conn.CloseNow()
	conn.SetReadLimit(4 << 20)
	c := searchCDP{conn: conn}
	raw, e := c.call(ctx, "Target.createBrowserContext", map[string]any{"disposeOnDetach": true}, "")
	if e != nil {
		return nil, e
	}
	var bc struct {
		ID string `json:"browserContextId"`
	}
	_ = json.Unmarshal(raw, &bc)
	u, _ := url.Parse(engine.URL)
	params := u.Query()
	params.Set("q", q)
	u.RawQuery = params.Encode()
	raw, e = c.call(ctx, "Target.createTarget", map[string]any{"url": u.String(), "browserContextId": bc.ID, "background": true}, "")
	if e != nil {
		return nil, e
	}
	var target struct {
		ID string `json:"targetId"`
	}
	_ = json.Unmarshal(raw, &target)
	raw, e = c.call(ctx, "Target.attachToTarget", map[string]any{"targetId": target.ID, "flatten": true}, "")
	if e != nil {
		return nil, e
	}
	var session struct {
		ID string `json:"sessionId"`
	}
	_ = json.Unmarshal(raw, &session)
	expression := `(()=>{const text=document.body?.innerText||'';const challenge=!!document.querySelector('iframe[src*="captcha"],input[name="captcha"],#captcha')||/unusual traffic|verify you are human|请完成验证|安全验证|人机验证/i.test(text);const items=[];for(const h of document.querySelectorAll('h2,h3')){const a=h.closest('a')||h.querySelector('a');if(a&&/^https?:/.test(a.href)&&h.innerText.trim())items.push({title:h.innerText.trim(),url:a.href,snippet:''})}return {challenge,results:items.slice(0,10)}})()`
	raised := false
	for {
		raw, e = c.call(ctx, "Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true}, session.ID)
		if e != nil {
			return nil, e
		}
		var result struct {
			Result struct {
				Value struct {
					Challenge bool        `json:"challenge"`
					Results   []webResult `json:"results"`
				} `json:"value"`
			} `json:"result"`
		}
		_ = json.Unmarshal(raw, &result)
		v := result.Result.Value
		if v.Challenge {
			if !human {
				return nil, fmt.Errorf("需要人工验证；可在搜索设置启用人工辅助验证")
			}
			if !raised {
				raised = true
				_, _ = c.call(ctx, "Target.activateTarget", map[string]any{"targetId": target.ID}, "")
				win, err := c.call(ctx, "Browser.getWindowForTarget", map[string]any{"targetId": target.ID}, "")
				if err == nil {
					var w struct {
						ID int `json:"windowId"`
					}
					_ = json.Unmarshal(win, &w)
					_, _ = c.call(ctx, "Browser.setWindowBounds", map[string]any{"windowId": w.ID, "bounds": map[string]any{"windowState": "normal"}}, "")
				}
			}
		} else if len(v.Results) > 0 {
			out := []webResult{}
			for _, r := range v.Results {
				r.URL = cleanWebURL(r.URL)
				if r.URL != "" {
					r.Engine = engine.ID + "（浏览器）"
					out = append(out, r)
				}
			}
			return out, nil
		}
		select {
		case <-ctx.Done():
			if raised {
				return nil, fmt.Errorf("人工验证未完成或已取消；未绕过验证码")
			}
			return nil, fmt.Errorf("浏览器保底超时或没有可读取的结果")
		case <-time.After(400 * time.Millisecond):
		}
	}
}

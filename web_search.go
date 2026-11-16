package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type searchEngine struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Enabled  bool   `json:"enabled"`
	NeedsKey bool   `json:"needs_key"`
	HasKey   bool   `json:"has_key"`
	Key      string `json:"key,omitempty"`
	ClearKey bool   `json:"clear_key,omitempty"`
}
type searchStrategy struct {
	Engines       []searchEngine `json:"engines"`
	Mode          string         `json:"mode"`
	Preset        string         `json:"preset"`
	Browser       bool           `json:"browser_fallback_enabled"`
	BrowserEngine string         `json:"browser_fallback_engine_id"`
	Human         bool           `json:"browser_human_assist_enabled"`
	Minimum       int            `json:"minimum"`
	Delay         int            `json:"delay_ms"`
}
type webResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Engine  string `json:"engine"`
}
type searchAttempt struct {
	Engine string `json:"engine"`
	Count  int    `json:"count"`
	MS     int64  `json:"ms"`
	Error  string `json:"error,omitempty"`
}

var searchSettingsMu sync.Mutex

func searchDefaults() searchStrategy {
	return searchStrategy{Mode: "priority", Preset: "mainland", Browser: true, BrowserEngine: "bing_cn", Minimum: 3, Delay: 500, Engines: []searchEngine{
		{ID: "bing_cn", Name: "Bing", URL: "https://cn.bing.com/search", Enabled: true}, {ID: "baidu", Name: "百度", URL: "https://www.baidu.com/s", Enabled: true}, {ID: "duckduckgo", Name: "DuckDuckGo", URL: "https://html.duckduckgo.com/html/", Enabled: true}, {ID: "google", Name: "Google（浏览器）", URL: "https://www.google.com/search", Enabled: true},
		{ID: "yuanbao", Name: "元宝搜索", URL: "https://api.wsa.cloud.tencent.com/SearchPro", NeedsKey: true}, {ID: "brave", Name: "Brave Search API", URL: "https://api.search.brave.com/res/v1/web/search", NeedsKey: true}, {ID: "serper", Name: "Serper / Google API", URL: "https://google.serper.dev/search", NeedsKey: true}, {ID: "tinyfish", Name: "TinyFish", URL: "https://api.search.tinyfish.ai", NeedsKey: true}, {ID: "tavily", Name: "Tavily", URL: "https://api.tavily.com/search", NeedsKey: true}, {ID: "maclaw_hub", Name: "MaClaw Hub / RapidSearch（需账户令牌）", URL: "https://hub.maclaw.top/searchproxy/search", NeedsKey: true}}}
}
func searchKeyRef(path string, e searchEngine) string {
	return fmt.Sprintf("websearch-%x", sha256.Sum256([]byte(filepath.Clean(path)+"|"+e.ID+"|"+e.URL)))
}
func (a *App) readSearch() (searchStrategy, error) {
	s := searchDefaults()
	db, e := a.workspaceDB()
	if e != nil {
		return s, e
	}
	defer db.Close()
	_, e = db.Exec(`CREATE TABLE IF NOT EXISTS app_setting(name TEXT PRIMARY KEY,value TEXT NOT NULL)`)
	if e != nil {
		return s, e
	}
	var raw string
	if db.QueryRow(`SELECT value FROM app_setting WHERE name='websearch'`).Scan(&raw) == nil {
		if e = json.Unmarshal([]byte(raw), &s); e != nil {
			return s, e
		}
	}
	for i := range s.Engines {
		k, e := secretGet(keyService, searchKeyRef(a.profiles.path, s.Engines[i]))
		s.Engines[i].HasKey = e == nil && k != ""
		s.Engines[i].Key = ""
	}
	return s, nil
}
func validateSearch(s searchStrategy) error {
	if !oneOf(s.Mode, "priority", "smart", "aggregate") || !oneOf(s.BrowserEngine, "bing_cn", "google") || s.Minimum < 1 || s.Minimum > 10 || s.Delay < 100 || s.Delay > 5000 {
		return fmt.Errorf("搜索策略参数无效")
	}
	known := map[string]searchEngine{}
	for _, e := range searchDefaults().Engines {
		known[e.ID] = e
	}
	seen := map[string]bool{}
	if len(s.Engines) != len(known) {
		return fmt.Errorf("搜索引擎列表不完整")
	}
	for _, e := range s.Engines {
		if _, ok := known[e.ID]; !ok || seen[e.ID] {
			return fmt.Errorf("未知或重复搜索引擎")
		}
		seen[e.ID] = true
		u, err := url.Parse(e.URL)
		if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")) {
			return fmt.Errorf("搜索接口须为 HTTPS（本机测试可用 HTTP），勿在地址填密钥或查询参数")
		}
	}
	return nil
}
func (a *App) webSearchAPI(w http.ResponseWriter, r *http.Request) {
	var x struct {
		Action   string         `json:"action"`
		Strategy searchStrategy `json:"strategy"`
		Query    string         `json:"query"`
		Engine   string         `json:"engine"`
	}
	if !a.read(w, r, &x) {
		return
	}
	if x.Action == "get" {
		s, e := a.readSearch()
		if e != nil {
			fail(w, 500, e.Error())
			return
		}
		jsonOut(w, 200, s)
		return
	}
	if x.Action == "save" {
		s := x.Strategy
		if e := validateSearch(s); e != nil {
			fail(w, 400, e.Error())
			return
		}
		searchSettingsMu.Lock()
		defer searchSettingsMu.Unlock()
		catalog := map[string]searchEngine{}
		for _, e := range searchDefaults().Engines {
			catalog[e.ID] = e
		}
		for i := range s.Engines {
			v := &s.Engines[i]
			v.Name = catalog[v.ID].Name
			v.NeedsKey = catalog[v.ID].NeedsKey
			ref := searchKeyRef(a.profiles.path, *v)
			if v.Key != "" {
				if e := secretSet(keyService, ref, v.Key); e != nil {
					fail(w, 500, "密钥无法安全保存，设置未更新")
					return
				}
			} else if v.ClearKey {
				_ = secretDelete(keyService, ref)
			}
			v.Key = ""
			v.ClearKey = false
			v.HasKey = false
		}
		db, e := a.workspaceDB()
		if e != nil {
			fail(w, 500, e.Error())
			return
		}
		defer db.Close()
		_, e = db.Exec(`CREATE TABLE IF NOT EXISTS app_setting(name TEXT PRIMARY KEY,value TEXT NOT NULL)`)
		if e == nil {
			b, _ := json.Marshal(s)
			_, e = db.Exec(`INSERT INTO app_setting VALUES('websearch',?) ON CONFLICT(name) DO UPDATE SET value=excluded.value`, string(b))
		}
		if e != nil {
			fail(w, 500, e.Error())
			return
		}
		out, e := a.readSearch()
		if e != nil {
			fail(w, 500, e.Error())
			return
		}
		jsonOut(w, 200, out)
		return
	}
	if x.Action != "search" && x.Action != "test" {
		fail(w, 400, "未知搜索操作")
		return
	}
	q := strings.TrimSpace(x.Query)
	if q == "" || len([]rune(q)) > 1000 {
		fail(w, 400, "请输入1000字以内的搜索词")
		return
	}
	s, e := a.readSearch()
	if e != nil {
		fail(w, 500, e.Error())
		return
	}
	if x.Action == "test" {
		found := false
		for i := range s.Engines {
			s.Engines[i].Enabled = s.Engines[i].ID == x.Engine
			found = found || s.Engines[i].Enabled
		}
		if !found {
			fail(w, 400, "未知搜索引擎")
			return
		}
		s.Browser = false
		s.Mode = "priority"
	}
	budget := 55 * time.Second
	if s.Human {
		budget = 150 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), budget)
	defer cancel()
	out, attempts := a.runWebSearch(ctx, s, q)
	jsonOut(w, 200, map[string]any{"results": out, "attempts": attempts, "notice": "公开网络搜索摘要，不是讲座库证据或已核验的比赛规则。"})
}
func (a *App) runWebSearch(ctx context.Context, s searchStrategy, q string) ([]webResult, []searchAttempt) {
	out := []webResult{}
	attempts := []searchAttempt{}
	seen := map[string]bool{}
	type answer struct {
		r []webResult
		a searchAttempt
	}
	accept := func(x answer) {
		attempts = append(attempts, x.a)
		for _, r := range x.r {
			if !seen[r.URL] && len(out) < 20 {
				seen[r.URL] = true
				out = append(out, r)
			}
		}
	}
	run := func(engine searchEngine, browser bool) answer {
		at := time.Now()
		limit := 7 * time.Second
		if browser {
			limit = 18 * time.Second
			if s.Human {
				limit = 120 * time.Second
			}
		}
		c, cancel := context.WithTimeout(ctx, limit)
		defer cancel()
		var r []webResult
		var e error
		if browser {
			r, e = a.browserSearch(c, engine, q, s.Human)
		} else {
			key, _ := secretGet(keyService, searchKeyRef(a.profiles.path, engine))
			r, e = searchHTTP(c, engine, q, key)
		}
		diag := searchAttempt{Engine: engine.ID, Count: len(r), MS: time.Since(at).Milliseconds()}
		if e != nil {
			diag.Error = e.Error()
		}
		return answer{r, diag}
	}
	engines := []searchEngine{}
	for _, e := range s.Engines {
		if e.Enabled {
			engines = append(engines, e)
		}
	}
	if s.Mode == "priority" {
		for _, e := range engines {
			accept(run(e, e.ID == "google"))
			if len(out) >= s.Minimum || ctx.Err() != nil {
				break
			}
		}
	} else {
		results := make(chan answer, len(engines))
		for i, e := range engines {
			go func(i int, e searchEngine) {
				if s.Mode == "smart" && i > 0 {
					timer := time.NewTimer(time.Duration(i*s.Delay) * time.Millisecond)
					defer timer.Stop()
					select {
					case <-ctx.Done():
						results <- answer{a: searchAttempt{Engine: e.ID, Error: "已取消"}}
						return
					case <-timer.C:
					}
				}
				results <- run(e, e.ID == "google")
			}(i, e)
		}
		for range engines {
			select {
			case x := <-results:
				accept(x)
			case <-ctx.Done():
				return out, attempts
			}
		}
	}
	if len(out) < s.Minimum && s.Browser && ctx.Err() == nil {
		for _, e := range s.Engines {
			if e.ID == s.BrowserEngine {
				accept(run(e, true))
				break
			}
		}
	}
	return out, attempts
}

var stripHTML = regexp.MustCompile(`(?s)<[^>]*>`)

func searchText(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(stripHTML.ReplaceAllString(s, " "))), " ")
}
func cleanWebURL(s string) string {
	u, e := url.Parse(html.UnescapeString(s))
	if e != nil {
		return ""
	}
	if strings.HasPrefix(s, "//") {
		u.Scheme = "https"
	}
	if v := u.Query().Get("uddg"); v != "" {
		return cleanWebURL(v)
	}
	if u.Scheme != "https" && u.Scheme != "http" || u.Hostname() == "" || u.User != nil {
		return ""
	}
	return u.String()
}
func searchHTTP(ctx context.Context, e searchEngine, q, key string) ([]webResult, error) {
	if e.NeedsKey && key == "" {
		return nil, fmt.Errorf("尚未配置此引擎的密钥或账户令牌")
	}
	u, err := url.Parse(e.URL)
	if err != nil {
		return nil, err
	}
	method := "GET"
	payload := map[string]any{}
	params := u.Query()
	headers := http.Header{"Accept": []string{"application/json,text/html"}, "User-Agent": []string{"TeachingAgents/0.4.3"}}
	switch e.ID {
	case "bing_cn", "duckduckgo":
		params.Set("q", q)
	case "baidu":
		params.Set("wd", q)
	case "brave":
		params.Set("q", q)
		params.Set("count", "8")
		headers.Set("X-Subscription-Token", key)
	case "tinyfish":
		params.Set("query", q)
		headers.Set("X-API-Key", key)
	case "serper":
		method = "POST"
		payload = map[string]any{"q": q, "num": 8}
		headers.Set("X-API-KEY", key)
	case "tavily":
		method = "POST"
		payload = map[string]any{"query": q, "max_results": 8}
		headers.Set("Authorization", "Bearer "+key)
	case "yuanbao":
		method = "POST"
		payload = map[string]any{"Query": q}
		headers.Set("Authorization", "Bearer "+strings.TrimPrefix(key, "Bearer "))
	case "maclaw_hub":
		method = "POST"
		payload = map[string]any{"query": q, "engine": "auto", "limit": 8, "content": true, "region": "cn", "fallback": true}
		headers.Set("Authorization", "Bearer "+key)
	default:
		return nil, fmt.Errorf("该引擎需要浏览器")
	}
	u.RawQuery = params.Encode()
	b, _ := json.Marshal(payload)
	var body io.Reader
	if method == "POST" {
		body = bytes.NewReader(b)
		headers.Set("Content-Type", "application/json")
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header = headers
	client := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return fmt.Errorf("重定向过多")
		}
		if req.URL.Host != via[0].URL.Host {
			req.Header.Del("Authorization")
			req.Header.Del("X-API-Key")
			req.Header.Del("X-Subscription-Token")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("搜索连接失败或超时")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	if err != nil {
		return nil, err
	}
	if oneOf(e.ID, "bing_cn", "baidu", "duckduckgo") {
		return searchHTMLResults(string(raw), e.ID), nil
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return nil, fmt.Errorf("搜索服务返回的不是 JSON")
	}
	return searchJSONResults(v, e.ID), nil
}
func searchJSONResults(v any, engine string) []webResult {
	out := []webResult{}
	seen := map[string]bool{}
	var walk func(any)
	walk = func(v any) {
		if len(out) >= 10 {
			return
		}
		switch x := v.(type) {
		case []any:
			for _, v := range x {
				walk(v)
			}
		case map[string]any:
			get := func(keys ...string) string {
				for _, k := range keys {
					if s, ok := x[k].(string); ok && s != "" {
						return s
					}
				}
				return ""
			}
			u := cleanWebURL(get("url", "link", "Url", "URL", "Link"))
			title := searchText(get("title", "Title", "name"))
			if u != "" && title != "" && !seen[u] {
				seen[u] = true
				snippet := searchText(get("snippet", "description", "content", "Snippet", "Content", "Summary"))
				rs := []rune(snippet)
				if len(rs) > 700 {
					snippet = string(rs[:700])
				}
				out = append(out, webResult{title, u, snippet, engine})
			}
			for _, k := range []string{"web", "organic", "results", "data", "Response", "WebPages", "Pages", "SearchResults"} {
				if child, ok := x[k]; ok {
					walk(child)
				}
			}
		}
	}
	walk(v)
	return out
}

var searchAnchor = regexp.MustCompile(`(?is)<a\b([^>]*?)href=["']([^"']+)["']([^>]*)>(.*?)</a>`)

func searchHTMLResults(raw, engine string) []webResult {
	out := []webResult{}
	seen := map[string]bool{}
	for _, m := range searchAnchor.FindAllStringSubmatchIndex(raw, -1) {
		attrs := raw[m[2]:m[3]] + raw[m[6]:m[7]]
		before := raw[max(0, m[0]-100):m[0]]
		if engine == "duckduckgo" && !strings.Contains(attrs, "result__a") {
			continue
		}
		if engine != "duckduckgo" && !strings.Contains(before, "<h2") && !strings.Contains(before, "<h3") {
			continue
		}
		u := cleanWebURL(raw[m[4]:m[5]])
		t := searchText(raw[m[8]:m[9]])
		if u == "" || t == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, webResult{t, u, "", engine})
		if len(out) >= 8 {
			break
		}
	}
	return out
}

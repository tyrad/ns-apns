package web

import (
	"context"
	"embed"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed templates/*.html
var tplFS embed.FS

var pages = template.Must(template.New("").ParseFS(tplFS, "templates/*.html"))

type Page struct {
	Flash        string
	Error        string
	TelegramUser string
	AppID        int
	AppHashSet   bool
	SessionPath  string
	SourceBot    string
	SourceID     int64
	TestBot      string
	TestID       int64
	Proxy        string
	LogLevel     string
	LogPath      string
	TokensText   string
	TokenCount   int
	Sandbox      bool
	APNsOn       bool
	HTTPAddr     string
	HasQR        bool
	Connecting   bool
	Refresh      bool
	Debug        bool
	PasswordSet  bool
}

type Form struct {
	AppID        int
	AppHash      string
	SessionPath  string
	SourceBot    string
	TestBot      string
	Proxy        string
	LogLevel     string
	LogPath      string
	DeviceTokens []string
	Sandbox      bool
	HTTPPassword  string
	ClearPassword bool
}

type Hooks struct {
	View     func() Page
	Save     func(ctx context.Context, in Form) error
	PushTest func(ctx context.Context, kind string) error
	QR       func() []byte
	Password func() string
	LogPath  string
}

func Listen(ctx context.Context, addr string, hooks Hooks, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		render(w, withDebug(hooks.View(), r))
	})
	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		handleIndexPost(w, r, hooks)
	})
	mux.HandleFunc("GET /qr.png", func(w http.ResponseWriter, r *http.Request) {
		if hooks.QR == nil {
			http.NotFound(w, r)
			return
		}
		png := hooks.QR()
		if len(png) == 0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(png)
	})
	mux.HandleFunc("GET /logs", func(w http.ResponseWriter, r *http.Request) {
		serveLogs(w, hooks.LogPath)
	})


	srv := &http.Server{
		Addr:              addr,
		Handler:           gate(mux, hooks),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shCtx)
	}()
	shown := addr
	if strings.HasPrefix(shown, ":") {
		shown = "127.0.0.1" + shown
	}
	log.Info("设置页已打开", "addr", "http://"+shown)
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func gate(next http.Handler, hooks Hooks) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			switch r.FormValue("op") {
			case "login":
				handleLogin(w, r, hooks)
				return
			case "logout":
				clearAuthCookie(w)
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
		pass := ""
		if hooks.Password != nil {
			pass = hooks.Password()
		}
		if !authorized(r, pass) {
			renderLogin(w, Page{})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleLogin(w http.ResponseWriter, r *http.Request, hooks Hooks) {
	pass := ""
	if hooks.Password != nil {
		pass = hooks.Password()
	}
	if passwordOK(r.FormValue("password"), pass) {
		if pass != "" {
			setAuthCookie(w, pass)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	renderLogin(w, Page{Error: "密码不对"})
}

func handleIndexPost(w http.ResponseWriter, r *http.Request, hooks Hooks) {
	if err := r.ParseForm(); err != nil {
		p := withDebug(hooks.View(), r)
		p.Error = err.Error()
		render(w, p)
		return
	}
	if r.FormValue("op") == "push-test" {
		kind := strings.TrimSpace(r.FormValue("kind"))
		if kind == "" {
			kind = "reply"
		}
		p := withDebug(hooks.View(), r)
		if err := hooks.PushTest(r.Context(), kind); err != nil {
			p.Error = err.Error()
		} else {
			p.Flash = "已发送测试推送（" + kind + "）"
		}
		render(w, p)
		return
	}
	in := Form{
		AppID:        atoi(r.FormValue("app_id")),
		AppHash:      r.FormValue("app_hash"),
		SessionPath:  r.FormValue("session_path"),
		SourceBot:    r.FormValue("source_bot"),
		TestBot:      r.FormValue("test_bot"),
		Proxy:        r.FormValue("proxy"),
		LogLevel:     r.FormValue("log_level"),
		LogPath:      r.FormValue("log_path"),
		DeviceTokens: splitLines(r.FormValue("device_tokens")),
		Sandbox:       r.FormValue("sandbox") == "1",
		HTTPPassword:  r.FormValue("http_password"),
		ClearPassword: r.FormValue("clear_password") == "1",
	}
	if err := hooks.Save(r.Context(), in); err != nil {
		p := withDebug(hooks.View(), r)
		p.Error = err.Error()
		render(w, p)
		return
	}
	if in.ClearPassword {
		clearAuthCookie(w)
	} else if hooks.Password != nil {
		if pw := hooks.Password(); pw != "" {
			setAuthCookie(w, pw)
		}
	}
	p := withDebug(hooks.View(), r)
	if p.TelegramUser != "" {
		p.Flash = "已保存。Bot / token 立即生效。"
	} else if p.HasQR {
		p.Flash = "请用手机 Telegram 扫下方二维码登录。"
	} else if p.Connecting {
		p.Flash = "正在连接 Telegram…"
	} else {
		p.Flash = "已保存。若已填 api_id / api_hash，页面会自动去连接。"
	}
	render(w, p)
}

func withDebug(p Page, r *http.Request) Page {
	p.Debug = isDebug(r)
	return p
}

func isDebug(r *http.Request) bool {
	q := r.URL.Query()
	if _, ok := q["debug"]; ok {
		return true
	}
	if r.FormValue("debug") != "" {
		return true
	}
	for _, v := range q[""] {
		if v == "debug" {
			return true
		}
	}
	return false
}

func render(w http.ResponseWriter, p Page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ExecuteTemplate(w, "index.html", p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func renderLogin(w http.ResponseWriter, p Page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pages.ExecuteTemplate(w, "login.html", p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func serveLogs(w http.ResponseWriter, path string) {
	if path == "" {
		http.Error(w, "未配置日志文件", http.StatusNotFound)
		return
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "还没有日志，先跑一会儿再下载", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="ns-apns.log"`)
	_, _ = io.Copy(w, f)
}

func atoi(s string) int {
	s = strings.TrimSpace(s)
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func splitLines(raw string) []string {
	raw = strings.ReplaceAll(raw, ",", "\n")
	raw = strings.ReplaceAll(raw, ";", "\n")
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

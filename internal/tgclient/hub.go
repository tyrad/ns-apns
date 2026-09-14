package tgclient

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/skip2/go-qrcode"

	"ns-apns/internal/apns"
	"ns-apns/internal/config"
	"ns-apns/internal/forward"
	"ns-apns/internal/settings"
	"ns-apns/internal/web"
)

// Hub 先开网页，保存 api 后在同一进程里连 Telegram；缺 session 就在页面出二维码。
type Hub struct {
	log    *slog.Logger
	parent context.Context

	mu      sync.Mutex
	cfg     config.Config
	client  *Client
	stopTG  context.CancelFunc
	qrPNG   []byte
	lastErr string
	busy    bool
}

func NewHub(cfg config.Config, log *slog.Logger) *Hub {
	return &Hub{cfg: cfg, log: log}
}

func (h *Hub) Run(ctx context.Context) error {
	h.parent = ctx
	h.ensureTelegram()
	return web.Listen(ctx, h.cfg.HTTPAddr, h.hooks(), h.log)
}

func (h *Hub) hooks() web.Hooks {
	return web.Hooks{
		View:     h.view,
		Save:     h.save,
		PushTest: h.pushTest,
		QR:       h.qrPNGCopy,
		Password: h.password,
		LogPath:  h.cfg.LogPath,
	}
}

func (h *Hub) password() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.client != nil {
		return h.client.cfg.HTTPPassword
	}
	return h.cfg.HTTPPassword
}

func (h *Hub) view() web.Page {
	h.mu.Lock()
	c := h.client
	qr := len(h.qrPNG) > 0
	busy := h.busy
	lastErr := h.lastErr
	cfg := h.cfg
	h.mu.Unlock()

	var p web.Page
	if c != nil {
		p = c.webView()
	} else {
		p = web.PageFromFile(cfg.Settings())
	}
	p.HTTPAddr = cfg.HTTPAddr
	p.HasQR = qr
	p.Connecting = busy && p.TelegramUser == "" && !qr
	p.Error = lastErr
	p.Refresh = qr || p.Connecting
	return p
}

func (h *Hub) save(ctx context.Context, in web.Form) error {
	h.mu.Lock()
	cur := h.cfg.Settings()
	if h.client != nil {
		cur = h.client.cfg.Settings()
	}
	h.mu.Unlock()

	s := web.MergeForm(cur, in)
	if err := settings.Save(h.cfg.SettingsPath, s); err != nil {
		return err
	}
	listen := h.cfg.HTTPAddr
	cfg := config.FromFile(s)
	cfg.SettingsPath = h.cfg.SettingsPath
	cfg.HTTPAddr = listen

	h.mu.Lock()
	h.cfg = cfg
	c := h.client
	h.lastErr = ""
	h.mu.Unlock()

	if c != nil && c.apiReady() {
		if err := c.applySettings(ctx, s); err != nil {
			return err
		}
	}
	h.ensureTelegram()
	return nil
}

func (h *Hub) pushTest(ctx context.Context, kind string) error {
	h.mu.Lock()
	c := h.client
	cfg := h.cfg
	if c != nil {
		cfg = c.cfg
	}
	log := h.log
	h.mu.Unlock()
	if c != nil {
		return c.webPushTest(ctx, kind)
	}
	pusher, err := apns.New(cfg, log)
	if err != nil {
		return err
	}
	if pusher == nil {
		return fmt.Errorf("还没有 device token，请先保存")
	}
	return pusher.Send(ctx, forward.TestEvent(kind))
}

func (h *Hub) qrPNGCopy() []byte {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.qrPNG) == 0 {
		return nil
	}
	out := make([]byte, len(h.qrPNG))
	copy(out, h.qrPNG)
	return out
}

func (h *Hub) ensureTelegram() {
	h.mu.Lock()
	cfg := h.cfg
	already := h.client != nil || h.busy
	parent := h.parent
	h.mu.Unlock()
	if parent == nil {
		return
	}
	if err := cfg.RequireTelegram(); err != nil {
		return
	}
	if already {
		return
	}

	c, err := New(cfg, h.log)
	if err != nil {
		h.mu.Lock()
		h.lastErr = err.Error()
		h.mu.Unlock()
		h.log.Error("无法启动 Telegram", "err", err)
		return
	}
	c.onQR = func(url string) {
		png, encErr := qrcode.Encode(url, qrcode.Medium, 240)
		h.mu.Lock()
		if encErr != nil {
			h.lastErr = encErr.Error()
		} else {
			h.qrPNG = png
			h.lastErr = ""
		}
		h.mu.Unlock()
	}
	c.onOnline = func() {
		h.mu.Lock()
		h.qrPNG = nil
		h.lastErr = ""
		h.mu.Unlock()
	}

	tgCtx, cancel := context.WithCancel(parent)
	h.mu.Lock()
	h.client = c
	h.stopTG = cancel
	h.busy = true
	h.qrPNG = nil
	h.lastErr = ""
	h.mu.Unlock()

	go func() {
		err := c.Connect(tgCtx)
		h.mu.Lock()
		h.busy = false
		h.qrPNG = nil
		if err != nil && parent.Err() == nil && tgCtx.Err() == nil {
			h.lastErr = err.Error()
			h.client = nil
			h.stopTG = nil
			h.log.Error("Telegram 断开", "err", err)
		}
		h.mu.Unlock()
	}()
}

func (c *Client) apiReady() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.api != nil
}

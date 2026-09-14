package tgclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/mdp/qrterminal/v3"

	"ns-apns/internal/apns"
	"ns-apns/internal/config"
	"ns-apns/internal/settings"
)

var errNeedLogin = errors.New("未登录或 session 已失效，请先运行: ns-apns login")

type Client struct {
	cfg        config.Config
	log        *slog.Logger
	raw        *telegram.Client
	dispatcher *tg.UpdateDispatcher
	loggedIn   qrlogin.LoggedIn

	mu       sync.Mutex
	api      *tg.Client
	pusher   *apns.Sender
	sourceID int64
	testID   int64
	selfName string
	onQR     func(url string)
	onOnline func()
	dedup    *deduper
}

func New(cfg config.Config, log *slog.Logger) (*Client, error) {
	path := cfg.SessionPath
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("创建 session 目录: %w", err)
		}
	}

	d := tg.NewUpdateDispatcher()
	dispatcher := &d
	c := &Client{
		cfg:        cfg,
		log:        log,
		dispatcher: dispatcher,
		loggedIn:   qrlogin.OnLoginToken(dispatcher),
		dedup:      &deduper{cap: 512},
	}
	opts := telegram.Options{
		SessionStorage: &telegram.FileSessionStorage{Path: path},
		UpdateHandler:  updateHook{next: dispatcher, c: c},
	}
	if r, err := resolverFromProxy(cfg.Proxy); err != nil {
		return nil, err
	} else if r != nil {
		opts.Resolver = r
	}

	pusher, err := apns.New(cfg, log)
	if err != nil {
		return nil, err
	}
	c.pusher = pusher
	c.raw = telegram.NewClient(cfg.AppID, cfg.AppHash, opts)
	return c, nil
}

func (c *Client) LoginQR(ctx context.Context) error {
	return mapAuthErr(c.raw.Run(ctx, func(ctx context.Context) error {
		if err := c.ensureQR(ctx); err != nil {
			return err
		}
		return c.printSelf(ctx, "登录成功")
	}))
}

func (c *Client) LoginPhone(ctx context.Context, phone string) error {
	flow := auth.NewFlow(terminalAuth{phone: phone}, auth.SendCodeOptions{})
	return mapAuthErr(c.raw.Run(ctx, func(ctx context.Context) error {
		if err := c.raw.Auth().IfNecessary(ctx, flow); err != nil {
			return fmt.Errorf("手机号登录: %w", err)
		}
		return c.printSelf(ctx, "登录成功")
	}))
}

func (c *Client) Connect(ctx context.Context) error {
	return mapAuthErr(c.raw.Run(ctx, func(ctx context.Context) error {
		if err := c.ensureQR(ctx); err != nil {
			return err
		}
		if err := c.printSelf(ctx, "已登录，开始监听"); err != nil {
			return err
		}
		if err := c.bindBots(ctx); err != nil {
			return err
		}
		<-ctx.Done()
		return ctx.Err()
	}))
}

func (c *Client) Run(ctx context.Context) error {
	return c.Connect(ctx)
}

func (c *Client) bindBots(ctx context.Context) error {
	botID, err := resolveUserID(ctx, c.raw.API(), c.cfg.SourceBot)
	if err != nil {
		return fmt.Errorf("解析 @%s: %w", c.cfg.SourceBot, err)
	}
	c.log.Info("已绑定来源", "bot", "@"+c.cfg.SourceBot, "id", botID)
	var testID int64
	if c.cfg.TestBot != "" {
		testID, err = resolveUserID(ctx, c.raw.API(), c.cfg.TestBot)
		if err != nil {
			c.log.Error("测试 Bot 解析失败，转发测试不可用", "bot", "@"+c.cfg.TestBot, "err", err)
			testID = 0
		} else {
			c.log.Info("测试转发已开", "bot", "@"+c.cfg.TestBot, "id", testID)
		}
	}
	c.mu.Lock()
	c.api = c.raw.API()
	c.sourceID = botID
	c.testID = testID
	c.mu.Unlock()
	return nil
}

func (c *Client) ensureQR(ctx context.Context) error {
	st, err := c.raw.Auth().Status(ctx)
	if err != nil {
		return err
	}
	if st.Authorized {
		return nil
	}
	fmt.Fprintln(os.Stderr, "\n请用手机 Telegram 扫描二维码：设置 → 设备 → 扫描二维码")
	show := func(ctx context.Context, token qrlogin.Token) error {
		qrterminal.GenerateHalfBlock(token.URL(), qrterminal.L, os.Stderr)
		fmt.Fprintf(os.Stderr, "或打开: %s\n等待扫描…\n", token.URL())
		if c.onQR != nil {
			c.onQR(token.URL())
		}
		return nil
	}
	if _, err := c.raw.QR().Auth(ctx, c.loggedIn, show); err != nil {
		if !tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
			return fmt.Errorf("扫码登录: %w", err)
		}
		pwd, err := terminalAuth{}.Password(ctx)
		if err != nil {
			return err
		}
		if _, err := c.raw.Auth().Password(ctx, pwd); err != nil {
			return fmt.Errorf("二步验证: %w", err)
		}
	}
	return nil
}

func (c *Client) printSelf(ctx context.Context, prefix string) error {
	self, err := c.raw.Self(ctx)
	if err != nil {
		return err
	}
	name := self.FirstName
	if self.Username != "" {
		name = fmt.Sprintf("%s (@%s)", name, self.Username)
	}
	c.mu.Lock()
	c.selfName = name
	c.mu.Unlock()
	c.log.Info(prefix, "user", name, "id", self.ID, "session", c.cfg.SessionPath)
	fmt.Fprintf(os.Stderr, "%s: %s\n", prefix, name)
	if c.onOnline != nil {
		c.onOnline()
	}
	return nil
}

func resolveUserID(ctx context.Context, api *tg.Client, username string) (int64, error) {
	res, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
		Username: settings.StripAt(username),
	})
	if err != nil {
		return 0, err
	}
	if u, ok := res.Peer.(*tg.PeerUser); ok {
		return u.UserID, nil
	}
	for _, x := range res.Users {
		u, ok := x.(*tg.User)
		if !ok {
			continue
		}
		if userHasUsername(u, username) {
			return u.ID, nil
		}
	}
	return 0, fmt.Errorf("不是用户")
}

func userHasUsername(u *tg.User, want string) bool {
	if strings.EqualFold(u.Username, want) {
		return true
	}
	for _, n := range u.Usernames {
		if strings.EqualFold(n.Username, want) {
			return true
		}
	}
	return false
}

func mapAuthErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, errNeedLogin) {
		return err
	}
	if tgerr.Is(err,
		"AUTH_KEY_UNREGISTERED",
		"SESSION_REVOKED",
		"SESSION_EXPIRED",
		"AUTH_KEY_DUPLICATED",
		"USER_DEACTIVATED",
		"USER_DEACTIVATED_BAN",
	) {
		return fmt.Errorf("%w (%v)", errNeedLogin, err)
	}
	return err
}

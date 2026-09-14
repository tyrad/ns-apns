package tgclient

import (
	"context"
	"fmt"
	"strings"

	"ns-apns/internal/apns"
	"ns-apns/internal/config"
	"ns-apns/internal/forward"
	"ns-apns/internal/settings"
	"ns-apns/internal/web"
)

func (c *Client) webView() web.Page {
	c.mu.Lock()
	defer c.mu.Unlock()
	return web.Page{
		TelegramUser: c.selfName,
		AppID:        c.cfg.AppID,
		AppHashSet:   c.cfg.AppHash != "",
		SessionPath:  c.cfg.SessionPath,
		SourceBot:    c.cfg.SourceBot,
		SourceID:     c.sourceID,
		TestBot:      c.cfg.TestBot,
		TestID:       c.testID,
		Proxy:        c.cfg.Proxy,
		LogLevel:     c.cfg.LogLevel,
		LogPath:      c.cfg.LogPath,
		TokensText:   strings.Join(c.cfg.DeviceTokens, "\n"),
		TokenCount:   len(c.cfg.DeviceTokens),
		APNsOn:       c.pusher != nil,
		HTTPAddr:     c.cfg.HTTPAddr,
		PasswordSet:  c.cfg.HTTPPassword != "",
	}
}

func (c *Client) applySettings(ctx context.Context, s settings.File) error {
	c.mu.Lock()
	api := c.api
	c.mu.Unlock()
	if api == nil {
		return fmt.Errorf("Telegram 尚未就绪")
	}

	s.Normalize()
	sourceID, err := resolveUserID(ctx, api, s.SourceBot)
	if err != nil {
		return fmt.Errorf("解析官方 Bot @%s: %w", s.SourceBot, err)
	}
	var testID int64
	if s.TestBot != "" {
		testID, err = resolveUserID(ctx, api, s.TestBot)
		if err != nil {
			c.log.Error("测试 Bot 解析失败，转发测试不可用", "bot", "@"+s.TestBot, "err", err)
			testID = 0
		}
	}

	listen := c.cfg.HTTPAddr
	cfg := config.FromFile(s)
	cfg.SettingsPath = c.cfg.SettingsPath
	cfg.HTTPAddr = listen
	pusher, err := apns.New(cfg, c.log)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.cfg = cfg
	c.sourceID = sourceID
	c.testID = testID
	c.pusher = pusher
	c.mu.Unlock()

	c.log.Info("设置已更新",
		"source", "@"+s.SourceBot,
		"test", "@"+s.TestBot,
		"devices", len(s.DeviceTokens),
	)
	return nil
}

func (c *Client) webPushTest(ctx context.Context, kind string) error {
	c.mu.Lock()
	pusher := c.pusher
	c.mu.Unlock()
	if pusher == nil {
		return fmt.Errorf("还没有 device token，请先保存")
	}
	return pusher.Send(ctx, forward.TestEvent(kind))
}

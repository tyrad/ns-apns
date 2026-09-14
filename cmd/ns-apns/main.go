package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"ns-apns/internal/config"
	"ns-apns/internal/tgclient"
	"ns-apns/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		usage()
		return
	}
	if cmd == "version" || cmd == "-v" || cmd == "--version" {
		fmt.Println(version.Version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log, logFile, err := newLogger(cfg.LogLevel, cfg.LogPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if logFile != nil {
		defer logFile.Close()
	}
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch cmd {
	case "test":
		err = runTest(os.Args[2:])
	case "push-test":
		err = runPushTest(ctx, cfg, log)
	case "login":
		if err = cfg.RequireTelegram(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		var c *tgclient.Client
		c, err = tgclient.New(cfg, log)
		if err != nil {
			log.Error(err.Error())
			os.Exit(1)
		}
		fs := flag.NewFlagSet("login", flag.ExitOnError)
		usePhone := fs.Bool("phone", false, "用手机号+验证码登录（默认扫码）")
		number := fs.String("number", "", "手机号")
		_ = fs.Parse(os.Args[2:])
		if *usePhone {
			err = c.LoginPhone(ctx, *number)
		} else {
			err = c.LoginQR(ctx)
		}
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		httpAddr := fs.String("http", "127.0.0.1:8787", "设置页监听地址")
		_ = fs.Parse(os.Args[2:])
		cfg.HTTPAddr = strings.TrimSpace(*httpAddr)
		if cfg.HTTPAddr == "" {
			cfg.HTTPAddr = "127.0.0.1:8787"
		}
		err = tgclient.NewHub(cfg, log).Run(ctx)
	default:
		usage()
		os.Exit(2)
	}

	if err != nil && ctx.Err() == nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

func newLogger(level, path string) (*slog.Logger, *os.File, error) {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	out := io.Writer(os.Stdout)
	var f *os.File
	if path != "" {
		var err error
		f, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return nil, nil, fmt.Errorf("打开日志文件: %w", err)
		}
		out = io.MultiWriter(os.Stdout, f)
	}
	h := slog.NewTextHandler(out, &slog.HandlerOptions{Level: lv})
	return slog.New(h), f, nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `ns-apns %s

监听 NodeSeek 官方 @nodemaid_bot 的评论通知并转发。
这是 Telegram 用户客户端，不是 Bot。session 等于你的账号登录态。

用法:
  ns-apns login            扫码登录（无桌面 VPS 可用）
  ns-apns login -phone     手机号+验证码登录
  ns-apns run [-http 127.0.0.1:8787]   打开设置页；保存 api 后可在网页扫码登录
  ns-apns test             不连 Telegram，模拟一条评论（测解析）
  ns-apns push-test        向已绑定的 device token 发一条测试推送
  ns-apns version

配置只在 settings.json。首次 run 会生成空文件，再在设置页填写。
设置页地址用 -http，默认 127.0.0.1:8787
`, version.Version)
}

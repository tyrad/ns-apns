package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"time"

	"ns-apns/internal/apns"
	"ns-apns/internal/config"
	"ns-apns/internal/forward"
	"ns-apns/internal/parser"
)

const sampleText = "ktieggboy评论了你的帖子, 点击查看 https://www.nodeseek.com/post-12345"

func runTest(args []string) error {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	text := fs.String("text", sampleText, "模拟官方 Bot 的评论文案")
	_ = fs.Parse(args)

	kind := parser.Classify(*text)
	if kind != parser.KindReply {
		return fmt.Errorf("不是评论文案（classify=%d），示例: %s", kind, sampleText)
	}
	parsed, ok := parser.ParseReply(*text, nil)
	if !ok {
		return fmt.Errorf("解析失败")
	}
	ev := forward.NewReply(0, time.Now(), *text, parsed)
	b, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func runPushTest(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pusher, err := apns.New(cfg, log)
	if err != nil {
		return err
	}
	if pusher == nil {
		return fmt.Errorf("未绑定 APNS_DEVICE_TOKEN")
	}
	ev := forward.NewReply(0, time.Now(), "ns-apns 测试推送", parser.Reply{
		Author: "ns-apns",
		URL:    "https://www.nodeseek.com/post-1",
		PostID: "1",
	})
	return pusher.Send(ctx, ev)
}

package apns

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"

	"ns-apns/internal/config"
	"ns-apns/internal/forward"
)

type Sender struct {
	mu             sync.Mutex
	primary        *apns2.Client
	fallback       *apns2.Client
	preferSandbox  bool
	topic          string
	tokens         []string
	log            *slog.Logger
}

func New(cfg config.Config, log *slog.Logger) (*Sender, error) {
	if log == nil {
		log = slog.Default()
	}
	if len(cfg.DeviceTokens) == 0 {
		log.Info("未绑定 APNS_DEVICE_TOKEN，评论只写日志")
		return nil, nil
	}
	authKey, err := token.AuthKeyFromBytes([]byte(apnsPrivateKey))
	if err != nil {
		return nil, fmt.Errorf("解析内置 .p8: %w", err)
	}
	tok := &token.Token{AuthKey: authKey, KeyID: keyID, TeamID: teamID}
	prod := apns2.NewTokenClient(tok).Production()
	dev := apns2.NewTokenClient(tok).Development()
	primary, fallback := prod, dev
	if cfg.Sandbox {
		primary, fallback = dev, prod
		log.Info("APNs 优先沙盒（Xcode 调试包）；BadDeviceToken 时会改打生产")
	} else {
		log.Info("APNs 优先生产；BadDeviceToken 时会改打沙盒")
	}
	log.Info("APNs 已启用", "topic", bundleID, "devices", len(cfg.DeviceTokens), "sandbox", cfg.Sandbox)
	return &Sender{
		primary:       primary,
		fallback:      fallback,
		preferSandbox: cfg.Sandbox,
		topic:         bundleID,
		tokens:        cfg.DeviceTokens,
		log:           log,
	}, nil
}

func (s *Sender) Send(ctx context.Context, ev forward.Event) error {
	if s == nil {
		return nil
	}
	pl := payloadFrom(ev)
	s.mu.Lock()
	tokens := append([]string(nil), s.tokens...)
	s.mu.Unlock()
	var first error
	for _, dev := range tokens {
		n := &apns2.Notification{
			DeviceToken: dev,
			Topic:       s.topic,
			CollapseID:  collapseID(ev),
			PushType:    apns2.PushTypeAlert,
			Payload:     pl,
		}
		env, res, err := s.pushWithFallback(ctx, n)
		if err != nil {
			s.log.Error("APNs 请求失败", "device", tokenTail(dev), "err", err)
			if first == nil {
				first = err
			}
			continue
		}
		if res.StatusCode != 200 {
			s.log.Error("APNs 被拒绝",
				"device", tokenTail(dev),
				"status", res.StatusCode,
				"reason", res.Reason,
				"env", env,
			)
			if first == nil {
				first = fmt.Errorf("apns %s", res.Reason)
			}
			continue
		}
		s.log.Info("APNs 已发送",
			"device", tokenTail(dev),
			"author", ev.Author,
			"post_id", ev.PostID,
			"env", env,
		)
	}
	return first
}

func (s *Sender) pushWithFallback(ctx context.Context, n *apns2.Notification) (string, *apns2.Response, error) {
	firstName, secondName := "production", "sandbox"
	first, second := s.primary, s.fallback
	if s.preferSandbox {
		firstName, secondName = "sandbox", "production"
	}
	res, err := first.PushWithContext(ctx, n)
	if err != nil {
		return firstName, nil, err
	}
	if res.StatusCode == 200 {
		return firstName, res, nil
	}
	if res.Reason != apns2.ReasonBadDeviceToken || second == nil {
		return firstName, res, nil
	}
	s.log.Info("token 与当前环境不匹配，改打另一侧", "from", firstName, "to", secondName)
	res2, err := second.PushWithContext(ctx, n)
	if err != nil {
		return secondName, nil, err
	}
	return secondName, res2, nil
}

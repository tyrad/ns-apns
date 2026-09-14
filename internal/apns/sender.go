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
	mu        sync.Mutex
	primary   *apns2.Client
	fallback  *apns2.Client
	topic     string
	tokens    []string
	log       *slog.Logger
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
	log.Info("APNs 已启用", "topic", bundleID, "devices", len(cfg.DeviceTokens))
	return &Sender{
		primary:  apns2.NewTokenClient(tok).Production(),
		fallback: apns2.NewTokenClient(tok).Development(),
		topic:    bundleID,
		tokens:   cfg.DeviceTokens,
		log:      log,
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
	res, err := s.primary.PushWithContext(ctx, n)
	if err != nil {
		return "production", nil, err
	}
	if res.StatusCode == 200 {
		return "production", res, nil
	}
	if res.Reason != apns2.ReasonBadDeviceToken || s.fallback == nil {
		return "production", res, nil
	}
	s.log.Info("token 与当前环境不匹配，改打另一侧", "from", "production", "to", "sandbox")
	res2, err := s.fallback.PushWithContext(ctx, n)
	if err != nil {
		return "sandbox", nil, err
	}
	return "sandbox", res2, nil
}

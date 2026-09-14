package tgclient

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"

	"ns-apns/internal/forward"
	"ns-apns/internal/parser"
)

type updateHook struct {
	next telegram.UpdateHandler
	c    *Client
}

func (h updateHook) Handle(ctx context.Context, u tg.UpdatesClass) error {
	h.c.consumeUpdates(ctx, u)
	if h.next != nil {
		return h.next.Handle(ctx, u)
	}
	return nil
}

func (c *Client) consumeUpdates(ctx context.Context, u tg.UpdatesClass) {
	for _, msg := range messagesIn(u) {
		c.routeMessage(ctx, msg)
	}
}

func (c *Client) routeMessage(ctx context.Context, msg *tg.Message) {
	peer, ok := peerUserID(msg.PeerID)
	if !ok {
		return
	}
	c.mu.Lock()
	sourceID, testID := c.sourceID, c.testID
	c.mu.Unlock()
	switch {
	case sourceID != 0 && peer == sourceID && !msg.Out:
		c.handle(ctx, msg, false)
	case testID != 0 && peer == testID:
		c.handle(ctx, msg, true)
	}
}

func (c *Client) handle(ctx context.Context, msg *tg.Message, test bool) {
	if c.dedup != nil && !c.dedup.fresh(msg.ID) {
		return
	}
	text := strings.TrimSpace(msg.Message)
	if text == "" {
		return
	}
	if test {
		c.log.Info("测试 Bot 收到消息", "message_id", msg.ID, "out", msg.Out, "text", clip(text, 80))
	}

	kind, parsed := parser.Parse(text, entitiesOf(msg))
	at := time.Unix(int64(msg.Date), 0).In(time.Local)
	ev := forward.Event{
		Type:              parser.TypeName(kind),
		Source:            parser.Source,
		TelegramMessageID: msg.ID,
		OccurredAt:        at,
		Author:            parsed.Author,
		URL:               parsed.URL,
		PostID:            parsed.PostID,
		Floor:             parsed.Floor,
		RawText:           text,
	}

	label := ev.Type
	if test {
		label = "测试 " + ev.Type
	}
	c.log.Info(label,
		"message_id", ev.TelegramMessageID,
		"author", ev.Author,
		"url", ev.URL,
		"post_id", ev.PostID,
		"floor", ev.Floor,
		"raw", ev.RawText,
	)

	c.mu.Lock()
	pusher := c.pusher
	c.mu.Unlock()
	if pusher == nil {
		c.log.Warn("有通知但未绑定 device token，只写日志")
		return
	}
	go func() {
		if err := pusher.Send(ctx, ev); err != nil {
			c.log.Error("APNs 发送失败", "message_id", ev.TelegramMessageID, "err", err)
		}
	}()
}

func messagesIn(u tg.UpdatesClass) []*tg.Message {
	var updates []tg.UpdateClass
	switch v := u.(type) {
	case *tg.Updates:
		updates = v.Updates
	case *tg.UpdatesCombined:
		updates = v.Updates
	case *tg.UpdateShort:
		updates = []tg.UpdateClass{v.Update}
	default:
		return nil
	}
	var out []*tg.Message
	for _, up := range updates {
		switch x := up.(type) {
		case *tg.UpdateNewMessage:
			if m, ok := x.Message.(*tg.Message); ok {
				out = append(out, m)
			}
		case *tg.UpdateEditMessage:
			if m, ok := x.Message.(*tg.Message); ok {
				out = append(out, m)
			}
		}
	}
	return out
}

func peerUserID(p tg.PeerClass) (int64, bool) {
	u, ok := p.(*tg.PeerUser)
	if !ok {
		return 0, false
	}
	return u.UserID, true
}

func entitiesOf(msg *tg.Message) []parser.Entity {
	var out []parser.Entity
	for _, raw := range msg.Entities {
		switch e := raw.(type) {
		case *tg.MessageEntityTextURL:
			out = append(out, parser.Entity{Type: "text_link", Offset: e.Offset, Length: e.Length, URL: e.URL})
		case *tg.MessageEntityURL:
			out = append(out, parser.Entity{Type: "url", Offset: e.Offset, Length: e.Length})
		}
	}
	return out
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}

type deduper struct {
	mu    sync.Mutex
	seen  map[int]struct{}
	order []int
	cap   int
}

func (d *deduper) fresh(id int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.seen == nil {
		d.seen = make(map[int]struct{})
	}
	if _, ok := d.seen[id]; ok {
		return false
	}
	d.seen[id] = struct{}{}
	d.order = append(d.order, id)
	if len(d.order) > d.cap {
		old := d.order[0]
		d.order = d.order[1:]
		delete(d.seen, old)
	}
	return true
}

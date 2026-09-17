package forward

import (
	"time"

	"ns-apns/internal/parser"
)

type Event struct {
	Type              string    `json:"type"`
	Source            string    `json:"source"`
	TelegramMessageID int       `json:"telegram_message_id"`
	OccurredAt        time.Time `json:"occurred_at"`
	Author            string    `json:"author"`
	URL               string    `json:"url,omitempty"`
	PostID            string    `json:"post_id,omitempty"`
	Floor             string    `json:"floor,omitempty"`
	RawText           string    `json:"raw_text"`
}

func NewReply(msgID int, at time.Time, raw string, r parser.Reply) Event {
	return Event{
		Type:              "reply",
		Source:            parser.Source,
		TelegramMessageID: msgID,
		OccurredAt:        at,
		Author:            r.Author,
		URL:               r.URL,
		PostID:            r.PostID,
		Floor:             r.Floor,
		RawText:           raw,
	}
}

func TestEvent(kind string) Event {
	now := time.Now()
	switch kind {
	case "at", "mention":
		return Event{
			Type:       "at",
			Source:     parser.Source,
			OccurredAt: now,
			Author:     "ns-apns",
			URL:        "https://www.nodeseek.com/post-1-1#1",
			PostID:     "1",
			Floor:      "1",
			RawText:    "测试：有人提到了你",
		}
	case "checkin":
		return Event{
			Type:       "checkin",
			Source:     parser.Source,
			OccurredAt: now,
			RawText:    "测试：今天的签到收益是5个",
		}
	case "message", "dm":
		return Event{
			Type:       "message",
			Source:     parser.Source,
			OccurredAt: now,
			Author:     "ns-apns",
			URL:        "https://www.nodeseek.com/notification#/message?mode=talk&to=1",
			RawText:    "测试：一条私信",
		}
	case "inbox", "notification":
		return Event{
			Type:       "inbox",
			Source:     parser.Source,
			OccurredAt: now,
			RawText:    "测试：站内通知",
		}
	default:
		return NewReply(0, now, "测试：评论了你的帖子", parser.Reply{
			Author: "ns-apns",
			URL:    "https://www.nodeseek.com/post-1-1#1",
			PostID: "1",
			Floor:  "1",
		})
	}
}

package apns

import (
	"fmt"
	"strings"

	"github.com/sideshow/apns2/payload"

	"ns-apns/internal/forward"
)

func payloadFrom(ev forward.Event) *payload.Payload {
	title, body := alertCopy(ev)
	p := payload.NewPayload().
		AlertTitle(title).
		AlertBody(body).
		Sound("default").
		MutableContent()
	p.Custom("type", ev.Type)
	if ev.URL != "" {
		p.Custom("url", ev.URL)
	}
	if ev.PostID != "" {
		p.Custom("post_id", ev.PostID)
	}
	if ev.Floor != "" {
		p.Custom("floor", ev.Floor)
	}
	return p
}

func alertCopy(ev forward.Event) (title, body string) {
	author := ev.Author
	if author == "" {
		author = "有人"
	}
	body = ev.RawText
	if body == "" {
		body = ev.URL
	}
	switch ev.Type {
	case "at", "mention":
		if strings.Contains(ev.RawText, "@了我") {
			return "有用户@了你", body
		}
		return author + " @了你", body
	case "checkin":
		if body == "" {
			body = "今天的签到已完成"
		}
		return "签到结果", body
	case "message", "dm":
		return author + " 发来私信", body
	case "inbox", "notification":
		if body == "" {
			body = "有新的站内消息"
		}
		return "站内通知", body
	case "other", "unknown":
		if body == "" {
			body = "收到一条新通知"
		}
		return "NodeSeek 通知", body
	default:
		if ev.Author == "" {
			return "新评论", body
		}
		return ev.Author + " 评论了你的帖子", body
	}
}

func collapseID(ev forward.Event) string {
	if ev.TelegramMessageID == 0 {
		return ""
	}
	return fmt.Sprintf("ns-%d", ev.TelegramMessageID)
}

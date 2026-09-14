package apns

import (
	"encoding/json"
	"testing"

	"ns-apns/internal/forward"
)

func TestPayloadFrom(t *testing.T) {
	ev := forward.Event{
		Type:              "reply",
		TelegramMessageID: 37022,
		Author:            "atx",
		URL:               "https://www.nodeseek.com/post-758151-2#13",
		PostID:            "758151",
		Floor:             "13",
		RawText:           "atx评论了你的帖子，点击查看",
	}
	raw, err := payloadFrom(ev).MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	aps := got["aps"].(map[string]any)
	alert := aps["alert"].(map[string]any)
	if alert["title"] != "atx 评论了你的帖子" {
		t.Fatalf("title=%v", alert["title"])
	}
	if alert["body"] != ev.RawText {
		t.Fatalf("body=%v", alert["body"])
	}
	if got["url"] != ev.URL || got["post_id"] != "758151" || got["floor"] != "13" {
		t.Fatalf("custom=%v", got)
	}
	if collapseID(ev) != "ns-37022" {
		t.Fatalf("collapse=%s", collapseID(ev))
	}
}

func TestAlertCopyByType(t *testing.T) {
	title, _ := alertCopy(forward.Event{Type: "at", Author: "bob"})
	if title != "bob @了你" {
		t.Fatalf("at title=%q", title)
	}
	title, _ = alertCopy(forward.Event{Type: "at", Author: "有用户", RawText: "有用户@了我，点击查看"})
	if title != "有用户@了你" {
		t.Fatalf("official at title=%q", title)
	}
	title, _ = alertCopy(forward.Event{Type: "checkin", RawText: "5个鸡腿"})
	if title != "签到结果" {
		t.Fatalf("checkin title=%q", title)
	}
	title, _ = alertCopy(forward.Event{Type: "message", Author: "bob"})
	if title != "bob 发来私信" {
		t.Fatalf("message title=%q", title)
	}
	var body string
	title, body = alertCopy(forward.Event{Type: "other", RawText: "奇怪的通知"})
	if title != "NodeSeek 通知" || body != "奇怪的通知" {
		t.Fatalf("other title=%q body=%q", title, body)
	}
}

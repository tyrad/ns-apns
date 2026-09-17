package parser

import "testing"

func TestParseUnifiesKinds(t *testing.T) {
	kind, parsed := Parse("有用户@了我，点击查看", []Entity{{
		Type: "text_link",
		URL:  "https://www.nodeseek.com/notification#/atMe",
	}})
	if kind != KindAt || parsed.URL == "" {
		t.Fatalf("kind=%v url=%q", kind, parsed.URL)
	}
	kind, _ = Parse("收到一条系统提醒点击查看", nil)
	if kind != KindInbox {
		t.Fatalf("inbox kind=%v", kind)
	}
	kind, _ = Parse("随便什么广告", nil)
	if TypeName(kind) != "other" {
		t.Fatalf("other=%v", kind)
	}
	kind, parsed = Parse("ICMP不可达喵给你发了一条私信，点击查看", []Entity{{
		Type: "text_link",
		URL:  "https://www.nodeseek.com/notification#/message?mode=talk&to=28302",
	}})
	if kind != KindMessage || TypeName(kind) != "message" {
		t.Fatalf("message kind=%v name=%s", kind, TypeName(kind))
	}
	if parsed.Author != "ICMP不可达喵" {
		t.Fatalf("message author=%q", parsed.Author)
	}
	if parsed.URL != "https://www.nodeseek.com/notification#/message?mode=talk&to=28302" {
		t.Fatalf("message url=%q", parsed.URL)
	}
	if parsed.PostID != "" || parsed.Floor != "" {
		t.Fatalf("message must not take talk id as post, got %+v", parsed)
	}
}

func TestClassifyAt(t *testing.T) {
	if Classify("alice提到了你") != KindAt {
		t.Fatal("expected mention")
	}
	if Classify("有用户@了我，点击查看") != KindAt {
		t.Fatal("expected official at-me")
	}
	if Classify("有用户@我，点击查看") != KindAt {
		t.Fatal("expected short at-me")
	}
}

func TestParseAtOfficial(t *testing.T) {
	text := "有用户@了我，点击查看"
	prefix := "有用户@了我，"
	ent := []Entity{{
		Type:   "text_link",
		Offset: len([]rune(prefix)),
		Length: len([]rune("点击查看")),
		URL:    "https://www.nodeseek.com/post-758151-2#13",
	}}
	r, ok := ParseAt(text, ent)
	if !ok {
		t.Fatal("expected at")
	}
	if r.Author != "有用户" {
		t.Fatalf("author=%q", r.Author)
	}
	if r.PostID != "758151" || r.Floor != "13" {
		t.Fatalf("post=%q floor=%q url=%q", r.PostID, r.Floor, r.URL)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in   string
		want Kind
	}{
		{"ktieggboy评论了你的帖子, 点击查看", KindReply},
		{"爱吃肉的棒男孩评论了你的帖子, 点击查看", KindReply},
		{"🍗🍗🍗\n今天的签到收益是3个🍗，有点少", KindCheckin},
		{"今天的签到收益是10个🍗，欧皇认定", KindCheckin},
		{"收到一条系统提醒点击查看", KindInbox},
		{"ICMP不可达喵给你发了一条私信，点击查看", KindMessage},
		{"alice给你发了一条私信", KindMessage},
		{"U9 迎新有礼", KindOther},
		{"", KindOther},
	}
	for _, tc := range cases {
		if got := Classify(tc.in); got != tc.want {
			t.Fatalf("Classify(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseReplyWithTextLink(t *testing.T) {
	text := "ktieggboy评论了你的帖子, 点击查看"
	// 「点击查看」在 UTF-16 里的起点：前面都是 BMP，字节/符一致。
	prefix := "ktieggboy评论了你的帖子, "
	off := len([]rune(prefix))
	ent := []Entity{{
		Type:   "text_link",
		Offset: off,
		Length: len([]rune("点击查看")),
		URL:    "https://www.nodeseek.com/post-12345-2",
	}}
	r, ok := ParseReply(text, ent)
	if !ok {
		t.Fatal("expected reply")
	}
	if r.Author != "ktieggboy" {
		t.Fatalf("author=%q", r.Author)
	}
	if r.URL != "https://www.nodeseek.com/post-12345-2" {
		t.Fatalf("url=%q", r.URL)
	}
	if r.PostID != "12345" || r.Floor != "" {
		t.Fatalf("page must not be floor, post=%q floor=%q", r.PostID, r.Floor)
	}
}

func TestParseReplyFloorFromHash(t *testing.T) {
	text := "爱吃肉的棒男孩评论了你的帖子, 点击查看"
	ent := []Entity{{
		Type: "text_link",
		URL:  "https://www.nodeseek.com/post-924065-1#2",
	}}
	r, ok := ParseReply(text, ent)
	if !ok {
		t.Fatal("expected reply")
	}
	if r.PostID != "924065" || r.Floor != "2" {
		t.Fatalf("post=%q floor=%q url=%q", r.PostID, r.Floor, r.URL)
	}
}

func TestParseReplyWithoutURLStillOK(t *testing.T) {
	r, ok := ParseReply("爱吃肉的棒男孩评论了你的帖子, 点击查看", nil)
	if !ok {
		t.Fatal("expected reply")
	}
	if r.Author != "爱吃肉的棒男孩" {
		t.Fatalf("author=%q", r.Author)
	}
	if r.URL != "" || r.PostID != "" {
		t.Fatalf("expected empty url, got %+v", r)
	}
}

func TestParseReplyURLEntityUTF16(t *testing.T) {
	text := "x评论了你的帖子 https://www.nodeseek.com/post-9"
	idx := len([]rune("x评论了你的帖子 "))
	link := "https://www.nodeseek.com/post-9"
	ent := []Entity{{
		Type:   "url",
		Offset: idx,
		Length: len([]rune(link)),
	}}
	r, ok := ParseReply(text, ent)
	if !ok {
		t.Fatal("expected reply")
	}
	if r.PostID != "9" {
		t.Fatalf("post=%q url=%q", r.PostID, r.URL)
	}
}

func TestParseReplyPrefersNodeSeekLink(t *testing.T) {
	text := "bob评论了你的帖子, 点击查看"
	ent := []Entity{
		{Type: "text_link", URL: "https://example.com/ad"},
		{Type: "text_link", URL: "https://www.nodeseek.com/post-77"},
	}
	r, ok := ParseReply(text, ent)
	if !ok {
		t.Fatal("expected reply")
	}
	if r.PostID != "77" {
		t.Fatalf("post=%q url=%q", r.PostID, r.URL)
	}
}

func TestParseReplyNotComment(t *testing.T) {
	if _, ok := ParseReply("今天的签到收益是3个🍗，有点少", nil); ok {
		t.Fatal("checkin must not parse as reply")
	}
}

func TestParseMessageOfficial(t *testing.T) {
	text := "ICMP不可达喵给你发了一条私信，点击查看"
	ent := []Entity{{
		Type: "text_link",
		URL:  "https://www.nodeseek.com/notification#/message?mode=talk&to=28302",
	}}
	r, ok := ParseMessage(text, ent)
	if !ok {
		t.Fatal("expected message")
	}
	if r.Author != "ICMP不可达喵" {
		t.Fatalf("author=%q", r.Author)
	}
	if r.URL != "https://www.nodeseek.com/notification#/message?mode=talk&to=28302" {
		t.Fatalf("url=%q", r.URL)
	}
}

func TestParseMessageWithoutURLStillOK(t *testing.T) {
	r, ok := ParseMessage("bob给你发了一条私信，点击查看", nil)
	if !ok {
		t.Fatal("expected message")
	}
	if r.Author != "bob" {
		t.Fatalf("author=%q", r.Author)
	}
	if r.URL != "" {
		t.Fatalf("expected empty url, got %+v", r)
	}
}

func TestParseMessageNotInbox(t *testing.T) {
	if _, ok := ParseMessage("收到一条系统提醒点击查看", nil); ok {
		t.Fatal("inbox must not parse as message")
	}
}

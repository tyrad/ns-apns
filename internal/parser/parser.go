package parser

import (
	"net/url"
	"regexp"
	"strings"
	"unicode/utf16"
)

const Source = "nodeseek.telegram.nodemaid"

// Entity 对应 Telegram 文本实体。Offset/Length 是 UTF-16 码元，与客户端一致。
type Entity struct {
	Type   string // text_link 或 url
	Offset int
	Length int
	URL    string
}

type Reply struct {
	Author string
	URL    string
	PostID string
	Floor  string
}

type Kind int

const (
	KindOther Kind = iota
	KindReply
	KindCheckin
	KindAt
	KindInbox
)

var (
	replyRe    = regexp.MustCompile(`^(.+?)评论了你的帖子`)
	checkinRe  = regexp.MustCompile(`今天的签到收益是`)
	atMeRe     = regexp.MustCompile(`@了?我`)
	mentionRe  = regexp.MustCompile(`提到了你`)
	atAuthorRe = regexp.MustCompile(`^(.+?)提到了你`)
	inboxRe    = regexp.MustCompile(`系统提醒`)
	// /post-{id}-{page}#{floor} ，中间那段是分页不是楼层。
	postRe = regexp.MustCompile(`(?i)(?:nodeseek\.com|deepflood\.com)/post-(\d+)(?:-(\d+))?(?:#(\d+))?`)
)

func TypeName(k Kind) string {
	switch k {
	case KindReply:
		return "reply"
	case KindAt:
		return "at"
	case KindCheckin:
		return "checkin"
	case KindInbox:
		return "inbox"
	default:
		return "other"
	}
}

// Parse 把官方 Bot 原文收成统一结构，listen 不再按类型拼字段。
func Parse(text string, entities []Entity) (kind Kind, parsed Reply) {
	text = strings.TrimSpace(text)
	kind = Classify(text)
	switch kind {
	case KindReply:
		if r, ok := ParseReply(text, entities); ok {
			return kind, r
		}
	case KindAt:
		if r, ok := ParseAt(text, entities); ok {
			return kind, r
		}
	}
	url := firstURL(text, entities)
	parsed.URL = url
	parsed.PostID, parsed.Floor = parsePost(url)
	return kind, parsed
}

func Classify(text string) Kind {
	text = strings.TrimSpace(text)
	if text == "" {
		return KindOther
	}
	if replyRe.MatchString(text) {
		return KindReply
	}
	if atMeRe.MatchString(text) || mentionRe.MatchString(text) {
		return KindAt
	}
	if checkinRe.MatchString(text) {
		return KindCheckin
	}
	if inboxRe.MatchString(text) {
		return KindInbox
	}
	return KindOther
}

// ParseReply 解析评论通知。不是评论时 ok=false。
// 没有链接时仍返回 author，URL/PostID 为空。
func ParseReply(text string, entities []Entity) (Reply, bool) {
	text = strings.TrimSpace(text)
	m := replyRe.FindStringSubmatch(text)
	if m == nil {
		return Reply{}, false
	}
	r := Reply{Author: strings.TrimSpace(m[1])}
	rawURL := firstURL(text, entities)
	if rawURL == "" {
		return r, true
	}
	r.URL = rawURL
	r.PostID, r.Floor = parsePost(rawURL)
	return r, true
}

func ExtractURL(text string, entities []Entity) string {
	return firstURL(text, entities)
}

// ParseAt 解析 @ 通知。官方原文是「有用户@了我，点击查看」或「有用户@我」。
func ParseAt(text string, entities []Entity) (Reply, bool) {
	text = strings.TrimSpace(text)
	if Classify(text) != KindAt {
		return Reply{}, false
	}
	author := "有用户"
	if m := atAuthorRe.FindStringSubmatch(text); len(m) > 1 {
		if name := strings.TrimSpace(m[1]); name != "" {
			author = name
		}
	}
	r := Reply{Author: author}
	rawURL := firstURL(text, entities)
	if rawURL == "" {
		return r, true
	}
	r.URL = rawURL
	r.PostID, r.Floor = parsePost(rawURL)
	return r, true
}

func firstURL(text string, entities []Entity) string {
	var fallback string
	for _, e := range entities {
		u := strings.TrimSpace(e.URL)
		if e.Type == "url" && u == "" {
			u = utf16Slice(text, e.Offset, e.Length)
		}
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if !strings.Contains(u, "://") {
			u = "https://" + u
		}
		if isNodeSeekPost(u) {
			return u
		}
		if fallback == "" {
			fallback = u
		}
	}
	if fallback != "" {
		return fallback
	}
	if loc := postRe.FindStringIndex(text); loc != nil {
		return "https://" + text[loc[0]:loc[1]]
	}
	return ""
}

func parsePost(raw string) (postID, floor string) {
	m := postRe.FindStringSubmatch(raw)
	if m == nil {
		if u, err := url.Parse(raw); err == nil {
			combined := u.Host + u.Path
			if u.Fragment != "" {
				combined += "#" + u.Fragment
			}
			m = postRe.FindStringSubmatch(combined)
		}
	}
	if m == nil {
		return "", ""
	}
	return m[1], m[3]
}

func isNodeSeekPost(u string) bool {
	return postRe.MatchString(u)
}

func utf16Slice(s string, offset, length int) string {
	if offset < 0 || length <= 0 {
		return ""
	}
	u := utf16.Encode([]rune(s))
	end := offset + length
	if offset > len(u) || end > len(u) {
		return ""
	}
	return string(utf16.Decode(u[offset:end]))
}

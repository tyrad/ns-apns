package apns

import "strings"

func parseDeviceTokens(raw string) []string {
	raw = strings.ReplaceAll(raw, ",", " ")
	raw = strings.ReplaceAll(raw, ";", " ")
	raw = strings.ReplaceAll(raw, "\n", " ")
	var out []string
	seen := map[string]struct{}{}
	for _, p := range strings.Fields(raw) {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func tokenTail(tok string) string {
	if len(tok) <= 4 {
		return "****"
	}
	return "…" + tok[len(tok)-4:]
}

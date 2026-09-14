package apns

import "testing"

func TestParseDeviceTokens(t *testing.T) {
	got := parseDeviceTokens("aaa, bbb\nbbb;ccc")
	if len(got) != 3 || got[0] != "aaa" || got[2] != "ccc" {
		t.Fatalf("%v", got)
	}
	if parseDeviceTokens("  ") != nil && len(parseDeviceTokens("  ")) != 0 {
		t.Fatalf("empty")
	}
}

func TestTokenTail(t *testing.T) {
	if tokenTail("0123456789abcdef") != "…cdef" {
		t.Fatalf("%s", tokenTail("0123456789abcdef"))
	}
}

package settings

import (
	"path/filepath"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	in := File{
		AppID:        1,
		AppHash:      "abc",
		SourceBot:    "@nodemaid_bot",
		TestBot:      "nsconnnectbot",
		DeviceTokens: []string{" aaa ", "aaa", "bbb"},
		Sandbox:      true,
	}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out.AppID != 1 || out.AppHash != "abc" {
		t.Fatalf("%+v", out)
	}
	if out.SourceBot != "nodemaid_bot" || out.TestBot != "nsconnnectbot" {
		t.Fatalf("%+v", out)
	}
	if len(out.DeviceTokens) != 2 || out.DeviceTokens[0] != "aaa" {
		t.Fatalf("tokens=%v", out.DeviceTokens)
	}
}

func TestStripAtRepeated(t *testing.T) {
	if StripAt("@@nodemaid_bot") != "nodemaid_bot" {
		t.Fatalf("%q", StripAt("@@nodemaid_bot"))
	}
	if StripAt(" @foo ") != "foo" {
		t.Fatalf("%q", StripAt(" @foo "))
	}
}

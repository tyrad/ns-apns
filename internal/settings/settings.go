package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const Path = "settings.json"

type File struct {
	AppID        int      `json:"app_id"`
	AppHash      string   `json:"app_hash"`
	SessionPath  string   `json:"session_path"`
	SourceBot    string   `json:"source_bot"`
	TestBot      string   `json:"test_bot"`
	Proxy        string   `json:"proxy"`
	LogLevel     string   `json:"log_level"`
	LogPath      string   `json:"log_path"`
	DeviceTokens []string `json:"device_tokens"`
	Sandbox      bool     `json:"sandbox"`
	HTTPPassword string   `json:"http_password"`
}

func Default() File {
	return File{
		SessionPath: "session.json",
		SourceBot:   "nodemaid_bot",
		LogLevel:    "info",
		LogPath:     "ns-apns.log",
	}
}

func Load(path string) (File, error) {
	if path == "" {
		path = Path
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, fmt.Errorf("settings.json: %w", err)
	}
	f.Normalize()
	return f, nil
}

func Save(path string, f File) error {
	if path == "" {
		path = Path
	}
	f.Normalize()
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (f *File) Normalize() {
	f.AppHash = strings.TrimSpace(f.AppHash)
	f.SessionPath = strings.TrimSpace(f.SessionPath)
	if f.SessionPath == "" {
		f.SessionPath = "session.json"
	}
	f.SourceBot = StripAt(f.SourceBot)
	if f.SourceBot == "" {
		f.SourceBot = "nodemaid_bot"
	}
	f.TestBot = StripAt(f.TestBot)
	f.Proxy = strings.TrimSpace(f.Proxy)
	f.LogLevel = strings.ToLower(strings.TrimSpace(f.LogLevel))
	if f.LogLevel == "" {
		f.LogLevel = "info"
	}
	f.LogPath = strings.TrimSpace(f.LogPath)
	if f.LogPath == "" {
		f.LogPath = "ns-apns.log"
	}
	f.HTTPPassword = strings.TrimSpace(f.HTTPPassword)
	seen := map[string]struct{}{}
	var tokens []string
	for _, t := range f.DeviceTokens {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		tokens = append(tokens, t)
	}
	f.DeviceTokens = tokens
}

func StripAt(raw string) string {
	s := strings.TrimSpace(raw)
	for strings.HasPrefix(s, "@") {
		s = strings.TrimPrefix(s, "@")
	}
	return s
}

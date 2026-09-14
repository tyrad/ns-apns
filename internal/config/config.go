package config

import (
	"fmt"
	"os"

	"ns-apns/internal/settings"
)

type Config struct {
	AppID        int
	AppHash      string
	SessionPath  string
	SourceBot    string
	TestBot      string
	Proxy        string
	LogLevel     string
	DeviceTokens []string
	Sandbox      bool
	HTTPAddr     string
	SettingsPath string
	LogPath      string
	HTTPPassword string
}

func Load() (Config, error) {
	path := settings.Path
	f, err := settings.Load(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return Config{}, err
		}
		f = settings.Default()
		if err := settings.Save(path, f); err != nil {
			return Config{}, fmt.Errorf("写入 settings.json: %w", err)
		}
	}
	c := FromFile(f)
	c.SettingsPath = path
	return c, nil
}

func FromFile(f settings.File) Config {
	f.Normalize()
	return Config{
		AppID:        f.AppID,
		AppHash:      f.AppHash,
		SessionPath:  f.SessionPath,
		SourceBot:    f.SourceBot,
		TestBot:      f.TestBot,
		Proxy:        f.Proxy,
		LogLevel:     f.LogLevel,
		DeviceTokens: append([]string(nil), f.DeviceTokens...),
		Sandbox:      f.Sandbox,
		SettingsPath: settings.Path,
		LogPath:      f.LogPath,
		HTTPPassword: f.HTTPPassword,
	}
}

func (c Config) Settings() settings.File {
	return settings.File{
		AppID:        c.AppID,
		AppHash:      c.AppHash,
		SessionPath:  c.SessionPath,
		SourceBot:    c.SourceBot,
		TestBot:      c.TestBot,
		Proxy:        c.Proxy,
		LogLevel:     c.LogLevel,
		LogPath:      c.LogPath,
		DeviceTokens: append([]string(nil), c.DeviceTokens...),
		Sandbox:      c.Sandbox,
		HTTPPassword: c.HTTPPassword,
	}
}

func (c Config) RequireTelegram() error {
	if c.AppID <= 0 {
		return fmt.Errorf("settings.json 里还没有 app_id（my.telegram.org）")
	}
	if c.AppHash == "" {
		return fmt.Errorf("settings.json 里还没有 app_hash")
	}
	return nil
}

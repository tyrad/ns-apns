package web

import (
	"strings"

	"ns-apns/internal/settings"
)

func PageFromFile(f settings.File) Page {
	f.Normalize()
	return Page{
		AppID:       f.AppID,
		AppHashSet:  f.AppHash != "",
		SessionPath: f.SessionPath,
		SourceBot:   f.SourceBot,
		TestBot:     f.TestBot,
		Proxy:       f.Proxy,
		LogLevel:    f.LogLevel,
		LogPath:     f.LogPath,
		TokensText:  strings.Join(f.DeviceTokens, "\n"),
		TokenCount:  len(f.DeviceTokens),
		APNsOn:      len(f.DeviceTokens) > 0,
		PasswordSet: f.HTTPPassword != "",
	}
}

func MergeForm(cur settings.File, in Form) settings.File {
	if in.AppID > 0 {
		cur.AppID = in.AppID
	}
	if strings.TrimSpace(in.AppHash) != "" {
		cur.AppHash = in.AppHash
	}
	if in.SessionPath != "" {
		cur.SessionPath = in.SessionPath
	}
	cur.SourceBot = in.SourceBot
	cur.TestBot = in.TestBot
	cur.Proxy = in.Proxy
	if in.LogLevel != "" {
		cur.LogLevel = in.LogLevel
	}
	if in.LogPath != "" {
		cur.LogPath = in.LogPath
	}
	cur.DeviceTokens = in.DeviceTokens
	if in.ClearPassword {
		cur.HTTPPassword = ""
	} else if strings.TrimSpace(in.HTTPPassword) != "" {
		cur.HTTPPassword = in.HTTPPassword
	}
	cur.Normalize()
	return cur
}

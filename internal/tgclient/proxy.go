package tgclient

import (
	"fmt"
	"net/url"

	"github.com/gotd/td/telegram/dcs"
	"golang.org/x/net/proxy"
)

func resolverFromProxy(raw string) (dcs.Resolver, error) {
	if raw == "" {
		return nil, nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("TG_PROXY: %w", err)
	}
	switch u.Scheme {
	case "socks5", "socks5h":
	default:
		return nil, fmt.Errorf("TG_PROXY 目前只支持 socks5://，收到 %s", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("TG_PROXY 缺少 host:port")
	}
	var auth *proxy.Auth
	if u.User != nil {
		pass, _ := u.User.Password()
		auth = &proxy.Auth{User: u.User.Username(), Password: pass}
	}
	d, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
	if err != nil {
		return nil, err
	}
	cd, ok := d.(proxy.ContextDialer)
	if !ok {
		return nil, fmt.Errorf("socks5 不支持 DialContext")
	}
	return dcs.Plain(dcs.PlainOptions{Dial: cd.DialContext}), nil
}

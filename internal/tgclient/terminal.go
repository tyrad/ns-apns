package tgclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"golang.org/x/term"
)

// terminalAuth 交互式手机号登录。不支持注册新号。
type terminalAuth struct {
	phone string
}

func (a terminalAuth) Phone(_ context.Context) (string, error) {
	if strings.TrimSpace(a.phone) != "" {
		return strings.TrimSpace(a.phone), nil
	}
	fmt.Fprint(os.Stderr, "手机号（国际格式，例如 +8613800138000）: ")
	return readLine()
}

func (terminalAuth) Password(_ context.Context) (string, error) {
	fmt.Fprint(os.Stderr, "二步验证密码: ")
	if term.IsTerminal(int(syscall.Stdin)) {
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	return readLine()
}

func (terminalAuth) Code(_ context.Context, _ *tg.AuthSentCode) (string, error) {
	fmt.Fprint(os.Stderr, "验证码（已发到你的 Telegram）: ")
	return readLine()
}

func (terminalAuth) SignUp(context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("不支持用本程序注册新 Telegram 账号")
}

func (terminalAuth) AcceptTermsOfService(_ context.Context, tos tg.HelpTermsOfService) error {
	return &auth.SignUpRequired{TermsOfService: tos}
}

func readLine() (string, error) {
	s, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

# ns-apns

监听 NodeSeek 官方 Telegram「新提醒」（`@nodemaid_bot`），解析后推到 NS Connect。

这不是 Bot。官方把消息推到**你的 Telegram 账号**，本程序登录同一个号去收。

## 你需要先准备

部署前请自己申请一对 Telegram 接口凭证（这一步必须你本人登录，AI 做不了）：

1. 打开 [my.telegram.org/apps](https://my.telegram.org/apps)
2. 用已经绑定 NodeSeek「新提醒」的那个 Telegram 号登录
3. 新建应用，Platform 选 Desktop
4. 记下 `api_id` 和 `api_hash`，稍后填进设置页

这两项只标明「这是哪个程序」，**单凭它们登不上你的号**。不要发到群里；丢了再申请一对即可。

## 部署交给 AI

把服务器安装、设置页、扫码登录这些交给 AI，让它按这份文档做：

[`docs/deploy.md`](docs/deploy.md)

可以对 AI 说：

```
请按 docs/deploy.md 帮我部署 ns-apns。我已经申请好 Telegram 的 api_id 和 api_hash。
```

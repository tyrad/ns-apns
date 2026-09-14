# ns-apns 需求说明

日期：2026-09-14  
状态：已确认，待实现  
范围：自托管守护进程。用用户 Telegram 账号监听官方 `@nodemaid_bot`，**只转发评论**。

## 已确认

| # | 项 | 结论 |
|---|---|---|
| 1 | 官方 Bot | `@nodemaid_bot`（界面名「新提醒」） |
| 2 | P0 出口 | 先做转发（只写日志）。**不对接 APNs**，以后再做 |
| 3 | 交付 | Go 交叉编译二进制。**先不做 Docker** |
| 4 | 事件 | **只做评论转发**。签到、私信、@ 先不做 |
| 5 | 仓库 | 先私有 |

未列入上表的旧设想（签到解析、Docker、APNs、多事件类型）一律视为后期，不进 P0 代码。

## 1. 背景

NodeSeek 没有对第三方开放的通知推送 API。站内 `/api/notification/*` 是网页自己用的内部 XHR，不是契约。

官方实时通道只有：用户在「设置 / 联系方式」绑定 Telegram 后，服务端通过官方 Bot（界面名「新提醒」，早期称 `ns_maid`）把事件推到 **用户自己的 Telegram 账号**。

典型原文（来自真实对话框）：

```
🍗🍗🍗
今天的签到收益是3个🍗，有点少
```

```
ktieggboy评论了你的帖子, 点击查看
```

「点击查看」是 Telegram 文本链接，URL 在 message entity 里，不在纯文本里。对话框里会夹杂 Telegram 给 Bot 私聊塞的 `Ad`，需要丢掉。

本仓库的 iOS 客户端（NS Connect）已能轮询站内通知接口。ns-apns 要补的是：评论通知事件驱动转发，不轮询论坛。

签到彩蛋、APNs 进 App 都是后期，P0 不包含。

## 2. 目标（P0）

用户在自己的无桌面 VPS 上跑一个常驻进程：

1. 用 **用户自己的 Telegram 账号** 登录（和手机 Telegram 同类，不是 Bot）。
2. 只订阅 `@nodemaid_bot` 这一个对话。
3. 新消息以 Telegram update 推到这条长连接（不轮询聊天记录，不轮询 NodeSeek）。
4. **只解析评论**，过滤广告和其它文案。
5. 出口只写日志。不做 webhook。
6. 可选 `TG_TEST_BOT`：把符合格式的消息转到用户自己的 Bot，用于测 `run`（官方 Bot 仍只收对方发来的）。

交付形态：Go 交叉编译出的 linux/amd64、linux/arm64、本机可执行文件。用户自己部署。不做镜像、不做 compose。项目不托管任何人的 Telegram 登录态。

## 3. 非目标

- 不用 Bot API 去「监听另一个 Bot」。Telegram 不允许；官方 Bot 的 Token 也不在用户手里。
- 不轮询 `www.nodeseek.com` 的 Cookie / 内部 XHR。
- 不做多租户 SaaS（用户把验证码打到你的服务器）。
- 不在 iOS 进程里跑 Telegram 客户端。
- P0 不实现 APNs。事件 JSON 保持可扩展即可，不为 APNs 单开模块。
- P0 不解析签到、私信、@、收藏更新。这类消息记 debug 后丢弃，不当成功转发。
- 不提供 Docker / compose。
- 不替代手机上的官方 Telegram 通知。本程序是旁路，原对话框照常在。

## 4. 原理（必须写进文档，避免后续做成 Bot）

Telegram 有两套入口：

| | Bot API | 用户协议 MTProto |
|---|---|---|
| 身份 | Bot | 真人账号 |
| 能看见 | 发给这个 Bot 的消息 | 这个号能看见的对话 |
| 「新提醒」推到哪 | 推给用户，不推给第三方 Bot | 因此只能走这条 |

程序没有官方 Bot 的钥匙。它只是在 VPS 上再登一次用户的号，坐在「新提醒」对话框里收推送。

无桌面不构成障碍：不需要 Telegram Desktop / 浏览器窗口。登录一次后只留 session 文件，进程挂着收 update。

## 5. 为什么用 Go

可以用 Go 写，而且比 Python 用户客户端更适合「用户自己部署到无桌面 VPS」。

| 点 | 说明 |
|---|---|
| MTProto | [`gotd/td`](https://github.com/gotd/td) 是纯 Go 用户/Bot 客户端，支持验证码、二步验证、二维码登录、session 持久化、断线重连、update 恢复。空闲约 150KB / 连接。 |
| 部署 | 交叉编译出一个 linux/amd64 或 arm64 文件，VPS 不用装 Python 运行时。 |
| 登录 | 库依赖里已有二维码（`rsc.io/qr`），SSH 日志里打 ASCII QR，手机扫「设备」。 |
| 出口 | P0 只写日志；APNs 后期再加。 |
| 交付 | `go build` 出单文件，VPS 不用装运行时。 |

不要用 Bot 框架（telebot、gotgbot 等）做监听层。那些只能收发给「你的 Bot」的消息，看不见「新提醒」。

Bot API 只允许用在可选的「把解析结果再发到用户自己的 Bot」这种出口，不能当输入。

## 6. 用户部署模型

一人、一台机器、一个 Telegram 号。

用户准备：

1. 能直连 Telegram DC 的 VPS（连不上时允许配 SOCKS5 / MTProxy）。
2. 已经绑定 NodeSeek「新提醒」的那个 Telegram 号。
3. 自己在 [my.telegram.org](https://my.telegram.org) 申请的 `api_id` / `api_hash`。不要在仓库或镜像里内置一把公用钥匙。

项目提供：Go 源码、交叉编译说明、环境变量样例、登录说明。不提供 Docker。

数据盘只持久化 session。容器重建不能丢。session 等于这个号的完整登录态，文档必须写清风险。

## 7. 登录（无桌面）

按推荐顺序支持三种，至少实现 1 和 2：

1. **二维码**：进程在 stdout 打 QR，手机 Telegram → 设置 → 设备 → 扫描。
2. **本机先登录，拷 session 上 VPS**：笔记本跑一次 `ns-apns login`，把 session 文件 `scp` 到 volume。
3. **SSH 交互**：手机号 → 验证码（打到用户手机上的 Telegram）→ 若开启云密码再输入。

登录子命令结束后，常驻子命令只读 session，不再要 TTY。session 失效时进程退出非零，日志写明「需要重新登录」，不要死循环刷验证码。

## 8. 运行时行为

```
NodeSeek 站内事件
  → 官方「新提醒」推到用户账号
  → VPS 长连接收到 update
  → 过滤发送者（官方 Bot username，可配置）
  → 丢掉 Ad 和非评论文案
  → 解析评论（作者 + entity 里的链接）
  → 去重（Telegram message id）
  → 出口：日志
```

必须事件驱动：订阅 update，禁止定时 `getHistory` 当主路径。允许启动时用 history 做一次水位对齐，避免重启后把旧消息当新事件；默认只处理启动之后的新消息。

只监听一个 peer。不要把用户其它对话拉进处理逻辑。

## 9. 解析规则（P0 仅评论）

以官方 Bot 原文为准。

### 9.1 评论

样本：

```
ktieggboy评论了你的帖子, 点击查看
```

文本匹配：`^(.+)评论了你的帖子`

帖子 URL 从 message entities（`text_link` / `url`）取，禁止只扫纯文本。解析出 `post_id`（以及若有楼层）。

结构：

```json
{
  "type": "reply",
  "source": "nodeseek.telegram.nodemaid",
  "telegram_message_id": 456,
  "occurred_at": "2026-09-14T20:04:00+08:00",
  "author": "ktieggboy",
  "url": "https://www.nodeseek.com/post-xxxxx",
  "post_id": "xxxxx",
  "raw_text": "ktieggboy评论了你的帖子, 点击查看"
}
```

没有 entity URL 的评论：记 warn，仍转发 `author` 和原文，`url` / `post_id` 为空，不要丢。

### 9.2 丢弃（不转发）

- 发送者不是 `@nodemaid_bot`
- 带广告标记 / `Ad` 的消息
- 签到（`今天的签到收益是`）及其它非评论文案：debug 日志后丢弃

签到规则留到后期，P0 不要实现 flavor / legs。

## 10. 出口

P0 写日志。APNs 用 `sideshow/apns2`：`.p8` / Key ID / Team ID / Bundle ID 写死在程序里，用户只配 `APNS_DEVICE_TOKEN`。

### 后期 APNs（已定模型，本期不实现）

学 Bark：发推资格和「推到哪台手机」分开。

| 材料 | 谁持有 | 说明 |
|---|---|---|
| `.p8` + Key ID + Team ID + Bundle ID | 开发者，随 ns-apns 程序走 | 证明能以 NS Connect 名义连 APNs。用户不去苹果后台申请，也不填进 settings.json |
| device token | 用户 | 苹果经 **NS Connect App** 下发，不是用户手写一串。用户复制后绑到自己那份 ns-apns |

绑定：一份自托管进程只给配置里的 token 发推（可多台设备多条 token）。重装 App / 换机后 token 会变，要重新绑。

不要：

- 让用户用自己的苹果账号出 `.p8`（推不到 App Store 上的 NS Connect）
- 把 `.p8` 写进用户文档当必填配置
- 把 token 打进 info 日志或提交 git

残余风险与 Bark 相同：程序里的 `.p8` 可被抽出，拿到别人的 token 就能以 NS Connect 名义弹通知。门禁是 token 不要泄露。对个人自托管可接受。

评论事件字段已够组 APNs：`author` 标题，`raw` 正文，`url` / `post_id` / `floor` 深链。

## 11. 配置

环境变量 / 配置文件二选一，不要两者打架。建议：

| 项 | 必填 | 说明 |
|---|---|---|
| `TG_APP_ID` | 是 | my.telegram.org |
| `TG_APP_HASH` | 是 | my.telegram.org |
| `TG_SESSION_PATH` | 是 | session 文件路径 |
| `TG_SOURCE_BOT` | 否 | 默认 `@nodemaid_bot`，可改 |
| `TG_TEST_BOT` | 否 | 自己的测试 Bot，转发评论用 |
| `TG_PROXY` | 否 | socks5:// |
| `LOG_LEVEL` | 否 | 默认 info |

禁止把 session、api_hash、代理密码打到 info 日志。

## 12. 部署形态

最小：

```text
ns-apns login     # 二维码或手机号，写 session
ns-apns run       # 常驻，无 TTY
```

打包（P0）：

- `make build` 或脚本产出 `ns-apns`（当前 OS）以及 linux/amd64、linux/arm64
- 无 CGO
- 用户把二进制和 session 放到 VPS，systemd 或 `nohup` 自便，仓库不提供 Docker

说明必须覆盖：国内机房连不上 DC、session 失效、二步验证、无桌面二维码登录。

## 13. 安全与合规

- 产品边界：开源/发镜像，用户自己跑，开发者不经手 session。
- session 权限等同于该 Telegram 号，能读的不只是「新提醒」。文档用加粗写在登录步骤之前。
- 建议用户用已经绑定 NodeSeek 的号；不要引导「再注册一个号来收官方 Bot」——官方绑定的是具体 user id。
- Telegram 对用户协议挂机是灰区。定位为个人自用守护进程，不为用户集中托管。
- 不实现读取或转发用户其它对话的功能，减小误用面。

## 14. 分期

| 期 | 内容 | 完成标准 |
|---|---|---|
| P0 | Go 骨架、QR/拷贝 session 登录、长连接、只转发评论、日志、交叉编译 | `@nodemaid_bot` 推一条评论，日志出现解析结果 |
| 后期 | APNs：程序内置开发者 `.p8`，用户只绑 NS Connect 下发的 device token；签到/私信/@、Docker | 另开需求 |

## 15. 验收（P0）

1. 无桌面 Linux 上完成登录，不依赖图形界面。
2. 重启进程后无需再验证，session 仍有效。
3. `@nodemaid_bot` 推送评论样本，日志出现 `type=reply`，`author` 正确；有 entity 时 `url` 来自链接而非瞎猜。
4. 签到、广告不打评论日志。
5. 能交叉编译出 linux 二进制。
6. session 无效时进程退出非零，日志能看出要重新登录。

## 16. 与 NS Connect 的关系

本仓库独立、先私有。NS Connect 继续用站内接口做应用内通知列表。ns-apns P0 只把评论事件转出进程；接到 APNs / 深链另开需求。

两仓不要共享 Cookie，也不要在 iOS 里嵌入 gotd。

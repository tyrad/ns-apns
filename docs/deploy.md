# 准备与部署

监听 NodeSeek 官方 Telegram「新提醒」（`@nodemaid_bot`），解析后推到 NS Connect。这不是 Bot：官方把消息推到**你的 Telegram 账号**，本程序登录同一个号去收。

配置只有一份 `settings.json`，用网页改。没有 `.env`。仓库：https://github.com/tyrad/ns-apns （私有）。

## 谁做什么

**AI 做：** 问清机器和代理 → 拿到二进制 → 放到工作目录 → 用 systemd 拉起（不加 `Restart=`）→ 确认设置页能打开。做到这一步就停。

**人做：** 论坛绑电报、填 `api_id` / `api_hash`、手机扫码、在设置页自己贴 Device Token、可选测试 Bot。  
Device Token **不要发给 AI**，只贴到设置页。

**缺这些先问，不要猜：** 部署到哪台机器（SSH 用户和地址）、安装目录、这台机器能不能直连 Telegram（不行就要代理）。安装目录用提示词里的 `<安装目录>`，没填就问，不要自行当成 `/opt/ns-apns`。`api_id` / `api_hash` 人已经申请好的，可以给 AI 写进配置，也可以人自己在设置页填。

## 1. 数据安全（必读）

本程序会登录**你自己的 Telegram 号**。运行目录里的文件请当成账号密码保管：不要发给别人、不要传到网盘公开分享。

**`session.json`（最要紧）**  
扫码成功后程序会写出这份文件，意思是「这个号已经登录」。拿到它的人可以读这个号的对话、用这个号发消息，不只是看「新提醒」。

一旦怀疑泄露：打开手机 Telegram → 设置 → 设备，把这台踢掉；建议改密并打开两步验证。然后删掉旧的 `session.json`，再在网页重新扫码。

**`settings.json`**  
里面是你在设置页填的内容（Telegram 应用编号、手机推送 Token、访问密码等）。泄露后别人可能改你的配置、往你的手机发推。

**`ns-apns.log`**  
运行日志，可能带帖子链接。不要整份贴到网上。

**网页面板**  
默认只在本机打开（`127.0.0.1:8787`）。如果改成对外监听（启动时加 `-http 0.0.0.0:端口`），请先设访问密码。不设的话，谁打开页面都能改 Token、给你手机发通知。

备份 `settings.json` 和 `session.json` 时，备份盘也要你自己能管住。备份和原文件一样敏感。

## 2. 需要准备哪些东西

### 机器

- 一台能访问 Telegram 的 Linux（无桌面即可）。国内机房常连不上，需要本机已有的代理，例如 `socks5://127.0.0.1:1080`。
- 固定工作目录：提示词里的 `<安装目录>`。二进制、`settings.json`、`session.json`、日志都放这里。没提供就问。

架构对照（在目标机器上执行 `uname -m`）：

| `uname -m` | 下这个文件 |
|---|---|
| `x86_64` / `amd64` | `ns-apns-linux-amd64` |
| `aarch64` / `arm64` | `ns-apns-linux-arm64` |

### 要人自己去拿的

1. **NodeSeek 已绑定电报**  
   论坛「设置 / 联系方式」绑官方「新提醒」。后面扫码必须用**同一个** Telegram 号。

2. **Telegram `api_id` 和 `api_hash`**  
   打开 [my.telegram.org/apps](https://my.telegram.org/apps) 新建应用，Platform 选 Desktop。  
   它们只标明「这是哪个程序」，**单凭这两项登不上你的号**。不要发给别人；丢了再申请一对就行。

3. **Device Token（要把通知推到手机时才需要）**  
   打开 NS Connect → 设置 → 推送通知 → 获取 Token → 复制，**自己粘贴到设置页**。多台手机就每行一个。  
   换机或重装 App 后 Token 会变，要重新复制。不要发给 AI，也不要发到聊天里。

4. **访问密码（可选）**  
   空着 = 打开网页不用登录。网页可能被别人访问时再设。忘了密码：打开运行目录的 `settings.json`，改或清空 `http_password` 那一行。

5. **测试 Bot（可选，人自己搞）**  
   用 BotFather 建一个 Bot，设置页只填它的用户名，例如 `@xxx_bot`。先给这个 Bot 发一句 `/start`，再把官方「新提醒」原样转给它。  
   **不要把 BotFather 给的那串 token 填进来。**

### 程序会自己生成的

扫码或手机号登录后出现 `session.json`。这才是真正的登录状态。文件还在，重启程序通常不用再扫；被踢下线或文件丢了，才需要重新登录。

### 不用准备、也不要往里填的

- 推送到手机不需要再申请别的，设置页贴上 Token 即可。
- 官方提醒 Bot 默认就是 `@nodemaid_bot`，一般不用改。
- 不需要 Telegram Bot token，也不需要 `.env`。
- 不做 Docker。开机自启若需要，人自己 `systemctl enable`；不要加 Docker，也不要擅自加 `Restart=`。

## 3. 怎么部署

按顺序做。做到「设置页能打开」就停，把扫码和 Token 交还给人。

### 3.1 先问清

1. SSH 到哪台机器（用户、地址）。没说就问，不要当成本机。
2. 安装目录是哪。用提示词里的 `<安装目录>`，没填就问，不要猜。
3. `uname -m` 是什么。
4. 这台机器能不能直连 Telegram。连不上再问代理地址（常见 `socks5://127.0.0.1:1080`）。代理填进 `settings.json` 的 `proxy` 后**要重启进程**才生效。

### 3.2 拿到二进制（两种方式，选一种）

仓库是私有的，下载或克隆都要有这个仓库的权限。

**方式 A：从 Release 下载（不用编译）**

1. 浏览器打开 https://github.com/tyrad/ns-apns/releases （先登录 GitHub）。
2. 打开最新一条（或指定的 tag），按下表下对应文件。
3. 推荐在自己电脑上下好，再拷到服务器，服务器上不必登录 GitHub：

```bash
scp ns-apns-linux-amd64 user@你的VPS:<安装目录>/ns-apns
```

4. 若必须在服务器上直接拉，用 GitHub 个人访问令牌（PAT，权限能读这个私有仓），**不要用 `gh` 命令**：

```bash
# 把 TOKEN 换成令牌，文件名按架构改
curl -L \
  -H "Authorization: Bearer TOKEN" \
  -H "Accept: application/octet-stream" \
  -o ns-apns \
  https://github.com/tyrad/ns-apns/releases/download/v0.1.0/ns-apns-linux-amd64
chmod +x ns-apns
```

版本号改成 Release 页面上实际的 tag。令牌用完不要写进仓库、不要写进文档。

**方式 B：自己编译**

需要 Go 1.22+，以及克隆这个私有仓的权限（HTTPS 登录或 SSH key）。

```bash
git clone https://github.com/tyrad/ns-apns.git
cd ns-apns
make dist
```

产物在 `bin/ns-apns-linux-amd64` 或 `bin/ns-apns-linux-arm64`。本机调试用 `make build`。

### 3.3 放到服务器并拉起

```bash
sudo mkdir -p <安装目录>
sudo cp ns-apns-linux-amd64 <安装目录>/ns-apns   # 或 arm64 那个
sudo chmod +x <安装目录>/ns-apns
```

默认用 systemd，**不要加 `Restart=`**（除非人明确要求自动拉起）。SSH 断开后进程还在。

`/etc/systemd/system/ns-apns.service`：

```
[Unit]
Description=ns-apns
After=network-online.target

[Service]
Type=simple
WorkingDirectory=<安装目录>
ExecStart=<安装目录>/ns-apns run
# 改端口：ExecStart=<安装目录>/ns-apns run -http 127.0.0.1:8787
# 不要加 Restart=

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl start ns-apns
sudo systemctl status ns-apns --no-pager
```

人要开机自启时再执行 `sudo systemctl enable ns-apns`，AI 不要擅自 enable。

第一次启动会写出空的 `settings.json`。若人已经把 `api_id` / `api_hash`（以及可选的 `proxy`）给了 AI，可以写进 `<安装目录>/settings.json` 对应字段，**不要写 `device_tokens`**。然后 `sudo systemctl restart ns-apns`。

网页端口只看启动参数 `-http`，默认 `127.0.0.1:8787`，不在面板里改。不要擅自改成 `0.0.0.0`。

人在本地、程序在 VPS 时，用 SSH 隧道，不要把面板打到公网：

```bash
ssh -L 8787:127.0.0.1:8787 user@你的VPS
```

本机打开 http://127.0.0.1:8787/

### 3.4 设置页里人自己做完这些

AI 到这里停，把地址告诉人即可。

1. （可选）「访问密码」填上并保存。
2. 填 `api_id`、`api_hash`，保存。没有 `session.json` 时页面会出现二维码：手机 Telegram → 设置 → 设备 → 扫描。必须用绑定了 NodeSeek「新提醒」的那个号。
3. Device Token 自己从 NS Connect 复制，贴到本页。不要发给别人。
4. 测试 Bot 可选。官方 Bot 默认 `@nodemaid_bot`，不用改。
5. 顶部 Telegram 显示你的名字后即在监听。

Bot 名 / Token / 访问密码保存后立刻生效。改代理需重启进程。

命令行扫码仍可用（可选）：

```bash
./ns-apns login
./ns-apns login -phone -number +8613800138000
```

登录失效时进程按错误退出，**不会自动循环刷验证码**。重新扫码或看网页二维码。

### 3.5 验收（AI 做完 3.3 后检查这些）

- `systemctl is-active ns-apns` 为 `active`
- 在服务器上 `curl -sS -o /dev/null -w "%{http_code}" http://127.0.0.1:8787/` 能打开（有访问密码时可能是登录页，也算成功）
- 告诉人：本机用 SSH 隧道后打开 http://127.0.0.1:8787/ ，自己填 api、扫码、贴 Token

人扫码成功后：设置页顶部出现 Telegram 名字；日志 `ns-apns.log` 里能看到已登录、在监听。没扫码之前，没有名字不算失败。

### 更新与备份

- 更新：替换 `<安装目录>/ns-apns`，再 `sudo systemctl restart ns-apns`。`settings.json` 和 `session.json` 留在原目录。
- 备份：拷走 `settings.json` 和 `session.json`（含登录态，当机密保管）。
- session 还在，机器或进程重启后一般不用重新扫码。
- 被踢下线或文件丢失：删 `session.json`，再扫码。

## 4. 用测试 Bot 试推送（可选，人自己做）

1. BotFather 建一个 Bot，只要用户名
2. 手机先给它发 `/start`
3. 设置页填测试 Bot
4. 把官方「新提醒」原样转到这个 Bot

程序会按原文类型发推送（评论 / @ / 签到 / 系统提醒；认不出的也推，点进去不跳页）。

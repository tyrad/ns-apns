# 准备与部署

监听 NodeSeek 官方 Telegram「新提醒」（`@nodemaid_bot`），解析后推到 NS Connect。这不是 Bot：官方把消息推到**你的 Telegram 账号**，本程序登录同一个号去收。

配置只有一份 `settings.json`，用网页改。没有 `.env`。程序挂了要不要自动拉起、开机要不要自启，按你自己的习惯处理即可。

## 1. 数据安全（必读）

本程序会登录**你自己的 Telegram 号**。运行目录里的文件请当成账号密码保管：不要发给别人、不要丢进群、不要传到网盘公开分享。

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

都在设置页完成即可，一般不用手改文件。

### 机器

- 一台能访问 Telegram 的 Linux（无桌面即可）。国内机房常连不上，准备 `socks5://127.0.0.1:1080` 这类代理。
- 固定工作目录，例如 `/opt/ns-apns`。`settings.json`、`session.json`、日志都放这里。

### 要自己去拿、填进设置页的

1. **NodeSeek 已绑定电报**  
   论坛「设置 / 联系方式」绑官方「新提醒」。后面扫码必须用**同一个** Telegram 号。

2. **Telegram `api_id` 和 `api_hash`**  
   打开 [my.telegram.org/apps](https://my.telegram.org/apps) 新建应用，Platform 选 Desktop，把两个值填进设置页。  
   它们只标明「这是哪个程序」，**单凭这两项登不上你的号**。不要公开贴出来；丢了再申请一对就行。

3. **Device Token（要把通知推到手机时才需要）**  
   真机安装带推送的 NS Connect（模拟器通常没有 Token）。设置 → 推送通知 → 点进去申请权限 → 获取 Token → 复制，粘贴到设置页。多台手机就每行一个。  
   换机或重装 App 后 Token 会变，要重新复制。不要把 Token 发到群里或贴到公开地方。

4. **访问密码（可选）**  
   空着 = 打开网页不用登录。网页可能被别人访问时再设。忘了密码：打开运行目录的 `settings.json`，改或清空 `http_password` 那一行。

5. **测试 Bot（可选）**  
   用 BotFather 建一个 Bot，设置页只填它的用户名，例如 `@xxx_bot`。先给这个 Bot 发一句 `/start`，再把官方「新提醒」原样转给它，用来试推送。  
   **不要把 BotFather 给的那串 token 填进来。**

### 程序会自己生成的

扫码或手机号登录后出现 `session.json`。这才是真正的登录状态。文件还在，重启程序通常不用再扫；被踢下线或文件丢了，才需要重新登录。

### 不用准备、也不要往里填的

- 苹果推送用的证书已经做进程序，不用去苹果开发者后台申请，也不要往设置里填。
- 官方提醒 Bot 默认就是 `@nodemaid_bot`，一般不用改。
- 不需要 Telegram Bot token，也不需要 `.env`。
- 当前交付是单个二进制，不做 Docker。

**沙盒开关：** 只有用 Xcode 装到真机上的调试包才勾「APNs 沙盒」。TestFlight 或 App Store 安装的包不要勾。

## 3. 怎么部署

### 下载现成二进制

打 `v*` 标签后，GitHub Actions 会编译并挂到该 tag 的 Release 上。VPS 按架构取其中一个：

- `ns-apns-linux-amd64`（常见云主机）
- `ns-apns-linux-arm64`（ARM 机器）

私有仓库下载需要登录 GitHub。把文件改名为 `ns-apns`，放到工作目录并 `chmod +x`。

### 自己编译（有 Go 时）

需要 Go 1.22+。本机调试用 `make build`，交叉编译用 `make dist`。

```bash
git clone <本仓库>
cd ns-apns
make dist
```

得到 `bin/ns-apns-linux-amd64` 或 `bin/ns-apns-linux-arm64`。把对应架构的二进制和以后生成的 `settings.json`、`session.json` 放在同一工作目录。

### 放到服务器

```bash
mkdir -p /opt/ns-apns
cp bin/ns-apns-linux-amd64 /opt/ns-apns/ns-apns
chmod +x /opt/ns-apns/ns-apns
cd /opt/ns-apns
./ns-apns run
# 改端口或对所有网卡监听（请先设访问密码）：
# ./ns-apns run -http 0.0.0.0:8787
```

第一次会写出空的 `settings.json`。浏览器打开：

```
http://127.0.0.1:8787
```

若人在本地、程序在 VPS，用 SSH 隧道，不要把面板直接打到公网：

```bash
ssh -L 8787:127.0.0.1:8787 user@你的VPS
```

本机再打开 http://127.0.0.1:8787。

### 设置页里做完这些

1. （可选）「访问密码」填上并保存。空着则不用登录。忘了改 `settings.json` 的 `http_password`。
2. 填 `api_id`、`api_hash`，保存。有旧 `session.json` 会自动上线；没有则扫页面二维码（手机 Telegram → 设置 → 设备 → 扫描）。
3. 粘贴 device token（可选）。Xcode 包打开「APNs 沙盒」。
4. 测试 Bot 可选。官方 Bot 默认 `nodemaid_bot`，不用改。
5. 顶部 Telegram 显示你的名字后即在监听。

Bot / token / 访问密码保存后立刻生效。改代理需自行重启进程。网页端口只看启动参数 `-http`，不在面板里改。

调试推送（会改地址栏，用完可去掉）：http://127.0.0.1:8787/?debug

命令行扫码仍可用（可选）：

```bash
./ns-apns login
./ns-apns login -phone -number +8613800138000
```

登录失效时进程按错误退出，**不会自动循环刷验证码**。重新扫码或看网页二维码。

### 进程怎么管

用 systemd、supervisor 还是 `tmux`/`nohup`，自行决定。本项目不强制开机自启，也不指定失败后自动拉起。

若用 systemd，自行写 unit，**自行决定**要不要 `Restart=`。至少固定工作目录，例如：

```
[Service]
Type=simple
WorkingDirectory=/opt/ns-apns
ExecStart=/opt/ns-apns/ns-apns run
# 需要改端口时：ExecStart=/opt/ns-apns/ns-apns run -http 127.0.0.1:8787
# Restart=  不要照抄；需要自动拉起时再自己加
```

更新：替换 `ns-apns` 二进制，再按你的方式重启进程。`settings.json` 和 `session.json` 留在原目录。

### 更新与备份

- 备份：拷走 `settings.json` 和 `session.json`（含登录态，当机密保管）。
- session 还在，机器或进程重启后一般不用重新扫码。
- 被踢下线或文件丢失：删 `session.json`，再扫码。

## 4. 用测试 Bot 试推送

1. BotFather 建一个 Bot，只要用户名
2. 手机先给它发 `/start`
3. 设置页填测试 Bot
4. 把官方「新提醒」原样转到这个 Bot

程序会按原文类型发推送（评论 / @ / 签到 / 系统提醒；认不出的也推，点进去不跳页）。

## 5. 点开通知会去哪

| 原文 | 点通知 |
|---|---|
| `{谁}评论了你的帖子` | 帖子详情 |
| `有用户@了我` | 有帖子链接进详情；否则打开「提到我」页面 |
| 签到 | 签到页 |
| `收到一条系统提醒` | 隐藏论坛头部，打开该条对话 |
| 无法识别 | 只出横幅，不跳页 |

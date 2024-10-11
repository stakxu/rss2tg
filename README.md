# RSS to Telegram Bot 使用文档


### 1. 编译 Go 程序

首先，编译为二进制文件：

```bash
go build -o rss2tg main.go
```
### 2. 创建 systemd 服务文件

接下来，创建一个 systemd 服务文件来管理程序。将服务文件放在 `/etc/systemd/system/` 目录下。

创建一个名为 `rss2tg.service` 的文件：

```bash
sudo nano /etc/systemd/system/rss2tg.service
```

在文件中添加以下内容：

```ini
[Unit]
Description=RSS to Telegram Bot
After=network.target

[Service]
ExecStart=/root/rss2tg/rss2tg
WorkingDirectory=/root/rss2tg
Restart=always
RestartSec=5
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=rss2tg
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

### 3. 启用并启动服务

保存并关闭文件后，使用以下命令重新加载 systemd 配置，启用并启动服务：

```bash
# 重新加载 systemd 配置
sudo systemctl daemon-reload

# 启用服务，使其在系统启动时自动启动
sudo systemctl enable rss2tg.service

# 启动服务
sudo systemctl start rss2tg.service
```

### 4. 检查服务状态

你可以使用以下命令检查服务的状态：

```bash
sudo systemctl status rss2tg.service
```

如果服务正常运行，你应该会看到类似以下的输出：

```bash
● rss2tg.service - RSS to Telegram Bot
   Loaded: loaded (/etc/systemd/system/rss2tg.service; enabled; vendor preset: enabled)
   Active: active (running) since Mon 2024-10-10 12:14:26 UTC; 1min ago
 Main PID: 7686 (rss2tg)
    Tasks: 5 (limit: 4915)
   Memory: 10.0M
   CGroup: /system.slice/rss2tg.service
           └─7686  /root/rss2tg/rss2tg
```

### 5. 管理服务

你可以使用以下命令来管理服务：

- **停止服务**：

  ```bash
  sudo systemctl stop rss2tg.service
  ```

- **重启服务**：

  ```bash
  sudo systemctl restart rss2tg.service
  ```

- **查看服务日志**：

  ```bash
  sudo journalctl -u rss2tg.service
  ```

## 2. 程序使用说明

### 2.1 配置文件

程序支持通过 YAML 配置文件或环境变量进行配置。配置文件位于 `/app/config/config.yaml`。如果该文件不存在，程序将使用环境变量进行初始配置。
环境变量读取优先级高于配置文件。

配置文件示例：

```yaml
telegram:
  bot_token: "your_bot_token_here"
  users:
    - "user_id_1"
    - "user_id_2"
  channels:
    - "@channel_1"
    - "@channel_2"
#这里的tg配置优先级低于环境变量，如果填在这里，一分钟后才会读取此配置
rss:
  - url: "https://example.com/feed1.xml"
    interval: 300
    keywords:
      - "面板"
      - "keyword2"

    exclude_keywords:  # 新增不推送关键词
    - "哪吒面板"  
    - "测评"  


    group: "Group1"
  - url: "https://example.com/feed2.xml"
    interval: 600
    keywords:
      - "keyword3"
    group: "Group2"
```

### 2.2 Bot 使用方法及命令

Bot 支持以下命令：

- `/start` - 开始使用机器人
- `/help` - 获取帮助信息
- `/config` - 查看当前配置
- `/add` - 添加 RSS 订阅
- `/edit` - 编辑 RSS 订阅
- `/delete` - 删除 RSS 订阅
- `/list` - 列出所有 RSS 订阅
- `/stats` - 查看推送统计

### 2.3 添加 RSS 订阅

#### 方式一

1. 发送 `/add` 命令给 Bot。
2. 按提示输入 RSS 订阅的 URL。
3. 输入更新间隔（秒）。
4. 输入关键词（用逗号分隔，如果没有可以直接输入 1）。
5. 输入组名。

#### 方式二

在当前config目录下新建config.ymal，填入以下内容。

```yaml
rss:
- url: https://rss.nodeseek.com
  interval: 30
  keywords:
  - vps
  - 甲骨文
  - 免费
  group: NS论坛
- url: https://linux.do/latest.rss
  interval: 30
  keywords:
  - vps
  - 甲骨文
  - 免费
  - 龟壳
  group: LC论坛
```

***两种方式都可以，系统会每1分钟自动检测，即使动态更改生效。***

### 2.4 编辑 RSS 订阅

1. 发送 `/edit` 命令给 Bot。
2. 输入要编辑的 RSS 订阅编号。
3. 按提示修改 URL、更新间隔、关键词和组名。如果不需要修改某项，直接输入 1。

### 2.5 删除 RSS 订阅

1. 发送 `/delete` 命令给 Bot。
2. 输入要删除的 RSS 订阅编号。

### 2.6 查看订阅列表

发送 `/list` 命令给 Bot，查看当前所有 RSS 订阅。

### 2.7 查看推送统计

发送 `/stats` 命令给 Bot，查看今日和本周的推送数量。


## 故障排查

- 如果 Bot 无响应，请检查 Telegram Bot Token 是否正确。
- 如果无法接收消息，请确保已将您的用户 ID 添加到配置中。


如有其他问题，请参考项目的 GitHub 页面或提交 issue。

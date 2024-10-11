package bot

import (
    "fmt"
    "log"
    "strconv"
    "strings"
    "time"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "rss2telegram/internal/config"
    "rss2telegram/internal/storage"
    "rss2telegram/internal/stats"
)

type MessageHandler func(title, url, group string, pubDate time.Time, matchedKeywords []string) error

type Bot struct {
    api              *tgbotapi.BotAPI
    users            []int64
    channels         []string
    db               *storage.Storage
    config           *config.Config
    configFile       string
    stats            *stats.Stats
    userState        map[int64]string
    messageHandler   MessageHandler
    updateRSSHandler func()
}

func NewBot(token string, users []string, channels []string, db *storage.Storage, config *config.Config, configFile string, stats *stats.Stats) (*Bot, error) {
    api, err := tgbotapi.NewBotAPI(token)
    if err != nil {
        return nil, err
    }

    userIDs := make([]int64, len(users))
    for i, user := range users {
        userID, err := strconv.ParseInt(user, 10, 64)
        if err != nil {
            return nil, fmt.Errorf("无效的用户ID: %s", user)
        }
        userIDs[i] = userID
    }

    return &Bot{
        api:              api,
        users:            userIDs,
        channels:         channels,
        db:               db,
        config:           config,
        configFile:       configFile,
        stats:            stats,
        userState:        make(map[int64]string),
        updateRSSHandler: func() {}, // 初始化为空函数
    }, nil
}

func (b *Bot) SetMessageHandler(handler MessageHandler) {
    b.messageHandler = handler
}

func (b *Bot) SetUpdateRSSHandler(handler func()) {
    b.updateRSSHandler = handler
}

func (b *Bot) Start() {
    log.Println("机器人已启动")
    
    commands := []tgbotapi.BotCommand{
        {Command: "start", Description: "开始使用机器人"},
        {Command: "help", Description: "获取帮助信息"},
        {Command: "config", Description: "查看当前配置"},
        {Command: "add", Description: "添加RSS订阅"},
        {Command: "edit", Description: "编辑RSS订阅"},
        {Command: "delete", Description: "删除RSS订阅"},
        {Command: "list", Description: "列出所有RSS订阅"},
        {Command: "stats", Description: "查看推送统计"},
    }
    
    setMyCommandsConfig := tgbotapi.NewSetMyCommands(commands...)
    _, err := b.api.Request(setMyCommandsConfig)
    if err != nil {
        log.Printf("设置命令失败: %v", err)
    }

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := b.api.GetUpdatesChan(u)

    for update := range updates {
        if update.Message == nil {
            continue
        }

        userID := update.Message.From.ID
        chatID := update.Message.Chat.ID

        if update.Message.IsCommand() {
            switch update.Message.Command() {
            case "start":
                b.handleStart(chatID)
            case "help":
                b.handleHelp(chatID)
            case "config":
                b.handleConfig(chatID)
            case "add":
                b.handleAdd(chatID, userID)
            case "edit":
                b.handleEdit(chatID, userID)
            case "delete":
                b.handleDelete(chatID, userID)
            case "list":
                b.handleList(chatID)
            case "stats":
                b.handleStats(chatID)
            default:
                b.sendMessage(chatID, "未知命令，请使用 /help 查看可用命令。")
            }
        } else {
            b.handleUserInput(update.Message)
        }
    }
}

func (b *Bot) SendMessage(title, url, group string, pubDate time.Time, matchedKeywords []string) error {
    chinaLoc, _ := time.LoadLocation("Asia/Shanghai")
    pubDateChina := pubDate.In(chinaLoc)
    
    // 将匹配的关键词加粗
    boldKeywords := make([]string, len(matchedKeywords))
    for i, keyword := range matchedKeywords {
        boldKeywords[i] = "*" + keyword + "*"
    }
    
    text := fmt.Sprintf("*%s*\n📡  %s\n🔍  %s\n🏷️  *%s*\n🕒  *%s*", 
        title, 
        url, 
        strings.Join(boldKeywords, ", "), 
        group, 
        pubDateChina.Format("2006-01-02 15:04:05"))
    
    log.Printf("发送消息: %s", text)

    for _, userID := range b.users {
        msg := tgbotapi.NewMessage(userID, text)
        msg.ParseMode = "Markdown"
        if _, err := b.api.Send(msg); err != nil {
            log.Printf("发送消息给用户 %d 失败: %v", userID, err)
        } else {
            log.Printf("成功发送消息给用户 %d", userID)
            b.stats.IncrementMessageCount()
        }
    }

    for _, channel := range b.channels {
        msg := tgbotapi.NewMessageToChannel(channel, text)
        msg.ParseMode = "Markdown"
        if _, err := b.api.Send(msg); err != nil {
            log.Printf("发送消息到频道 %s 失败: %v", channel, err)
        } else {
            log.Printf("成功发送消息到频道 %s", channel)
            b.stats.IncrementMessageCount()
        }
    }

    return nil
}

func (b *Bot) reloadConfig() error {
    newConfig, err := config.Load(b.configFile)
    if err != nil {
        return err
    }
    b.config = newConfig
    return nil
}

func (b *Bot) handleStart(chatID int64) {
    b.sendMessage(chatID, "欢迎使用RSS订阅机器人！使用 /help 查看可用命令。")
}

func (b *Bot) handleHelp(chatID int64) {
    helpText := `可用命令：
/config - 查看当前配置
/add - 添加RSS订阅
/edit - 编辑RSS订阅
/delete - 删除RSS订阅
/list - 列出所有RSS订阅
/stats - 查看推送统计`
    b.sendMessage(chatID, helpText)
}

func (b *Bot) handleConfig(chatID int64) {
    if err := b.reloadConfig(); err != nil {
        b.sendMessage(chatID, "加载配置时出错：" + err.Error())
        return
    }
    b.sendMessage(chatID, b.getConfig())
}

func (b *Bot) handleAdd(chatID int64, userID int64) {
    b.userState[userID] = "add_url"
    b.sendMessage(chatID, "请输入要添加的RSS订阅URL：")
}

func (b *Bot) handleEdit(chatID int64, userID int64) {
    b.userState[userID] = "edit_index"
    b.sendMessage(chatID, "请输入要编辑的RSS订阅编号：")
}

func (b *Bot) handleDelete(chatID int64, userID int64) {
    b.userState[userID] = "delete"
    b.sendMessage(chatID, "请输入要删除的RSS订阅编号：")
}

func (b *Bot) handleList(chatID int64) {
    if err := b.reloadConfig(); err != nil {
        b.sendMessage(chatID, "加载配置时出错：" + err.Error())
        return
    }
    b.sendMessage(chatID, b.listSubscriptions())
}

func (b *Bot) handleStats(chatID int64) {
    b.sendMessage(chatID, b.getStats())
}

func (b *Bot) handleUserInput(message *tgbotapi.Message) {
    userID := message.From.ID
    chatID := message.Chat.ID
    text := message.Text

    switch b.userState[userID] {
    case "add_url":
        b.userState[userID] = "add_interval"
        b.config.RSS = append(b.config.RSS, config.RSSConfig{  // 使用 config.RSSConfig
            URL: text,
        })
        b.sendMessage(chatID, "请输入订阅的更新间隔（秒）：")

    case "add_interval":
        interval, err := strconv.Atoi(text)
        if err != nil {
            b.sendMessage(chatID, "无效的间隔时间，请输入一个整数。")
            return
        }
        b.config.RSS[len(b.config.RSS)-1].Interval = interval
        b.userState[userID] = "add_keywords"
        b.sendMessage(chatID, "请输入关键词（用逗号分隔，如果没有可以直接输入1）：")
    case "add_keywords":
        if text != "1" {
            keywords := strings.Split(text, ",")
            b.config.RSS[len(b.config.RSS)-1].Keywords = keywords
        }
        b.userState[userID] = "add_group"
        b.sendMessage(chatID, "请输入组名：")
    case "add_group":
        b.config.RSS[len(b.config.RSS)-1].Group = text
        delete(b.userState, userID)
        if err := b.config.Save(b.configFile); err != nil {
            b.sendMessage(chatID, "添加订阅成功，但保存配置失败。")
        } else {
            b.sendMessage(chatID, "成功添加RSS订阅。")
            b.updateRSSHandler()
        }
    case "edit_index":
        index, err := strconv.Atoi(text)
        if err != nil || index < 1 || index > len(b.config.RSS) {
            b.sendMessage(chatID, "无效的编号。请使用 /edit 重新开始。")
            delete(b.userState, userID)
            return
        }
        b.userState[userID] = fmt.Sprintf("edit_url_%d", index-1)
        b.sendMessage(chatID, fmt.Sprintf("当前URL为：%s\n请输入新的URL（如不修改请输入1）：", b.config.RSS[index-1].URL))
    case "delete":
        index, err := strconv.Atoi(text)
        if err != nil || index < 1 || index > len(b.config.RSS) {
            b.sendMessage(chatID, "无效的编号。请使用 /delete 重新开始。")
            delete(b.userState, userID)
            return
        }
        deletedRSS := b.config.RSS[index-1]
        b.config.RSS = append(b.config.RSS[:index-1], b.config.RSS[index:]...)
        if err := b.config.Save(b.configFile); err != nil {
            b.sendMessage(chatID, "删除订阅成功，但保存配置失败。")
        } else {
            b.sendMessage(chatID, fmt.Sprintf("成功删除订阅: %s", deletedRSS.URL))
            b.updateRSSHandler()
        }
        delete(b.userState, userID)
    default:
        if strings.HasPrefix(b.userState[userID], "edit_url_") {
            index, _ := strconv.Atoi(strings.TrimPrefix(b.userState[userID], "edit_url_"))
            if text != "1" {
                b.config.RSS[index].URL = text
            }
            b.userState[userID] = fmt.Sprintf("edit_interval_%d", index)
            b.sendMessage(chatID, fmt.Sprintf("当前间隔为：%d秒\n请输入新的间隔时间（秒）（如不修改请输入1）：", b.config.RSS[index].Interval))
        } else if strings.HasPrefix(b.userState[userID], "edit_interval_") {
            index, _ := strconv.Atoi(strings.TrimPrefix(b.userState[userID], "edit_interval_"))
            if text != "1" {
                interval, err := strconv.Atoi(text)
                if err != nil {
                    b.sendMessage(chatID, "无效的间隔时间，请输入一个整数。不修改请输入1。")
                    return
                }
                b.config.RSS[index].Interval = interval
            }
            b.userState[userID] = fmt.Sprintf("edit_keywords_%d", index)
            b.sendMessage(chatID, fmt.Sprintf("当前关键词为：%v\n请输入新的关键词（用逗号分隔，如不修改请输入1）：", b.config.RSS[index].Keywords))
        } else if strings.HasPrefix(b.userState[userID], "edit_keywords_") {
            index, _ := strconv.Atoi(strings.TrimPrefix(b.userState[userID], "edit_keywords_"))
            if text != "1" {
                b.config.RSS[index].Keywords = strings.Split(text, ",")
            }
            b.userState[userID] = fmt.Sprintf("edit_group_%d", index)
            b.sendMessage(chatID, fmt.Sprintf("当前组名为：%s\n请输入新的组名（如不修改请输入1）：", b.config.RSS[index].Group))
        } else if strings.HasPrefix(b.userState[userID], "edit_group_") {
            index, _ := strconv.Atoi(strings.TrimPrefix(b.userState[userID], "edit_group_"))
            if text != "1" {
                b.config.RSS[index].Group = text
            }
            delete(b.userState, userID)
            if err := b.config.Save(b.configFile); err != nil {
                b.sendMessage(chatID, "编辑订阅成功，但保存配置失败。")
            } else {
                b.sendMessage(chatID, "成功编辑RSS订阅。")
                b.updateRSSHandler()
            }
        }
    }
}

func (b *Bot) sendMessage(chatID int64, text string) {
    msg := tgbotapi.NewMessage(chatID, text)
    if _, err := b.api.Send(msg); err != nil {
        log.Printf("发送消息失败: %v", err)
    }
}

func (b *Bot) getConfig() string {
    config := "当前配置信息：\n"
    config += fmt.Sprintf("用户: %v\n", b.users)
    config += fmt.Sprintf("频道: %v\n", b.channels)
    config += "RSS订阅:\n"
    for i, rss := range b.config.RSS {
        config += fmt.Sprintf("%d. 📡  URL: %s\n   ⏱️  间隔: %d秒\n   🔑  关键词: %v\n   🏷️  组名: %s\n", i+1, rss.URL, rss.Interval, rss.Keywords, rss.Group)
    }
    return config
}

func (b *Bot) listSubscriptions() string {
    list := "当前RSS订阅列表:\n"
    for i, rss := range b.config.RSS {
        list += fmt.Sprintf("%d. 📡  URL: %s\n   ⏱️  间隔: %d秒\n   🔑  关键词: %v\n   🏷️  组名: %s\n", i+1, rss.URL, rss.Interval, rss.Keywords, rss.Group)
    }
    return list
}

func (b *Bot) getStats() string {
    dailyCount, weeklyCount := b.stats.GetMessageCounts()
    return fmt.Sprintf("推送统计:\n📊  今日推送: %d\n📈  本周推送: %d", dailyCount, weeklyCount)
}

func (b *Bot) UpdateConfig(cfg *config.Config) {
    b.config = cfg
}

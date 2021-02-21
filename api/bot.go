package api

import (
	"fmt"
	"io"
	"log"
	"net/http"

	tgBot "github.com/go-telegram-bot-api/telegram-bot-api"
)

type (
	Client interface {
		Send(string, int64) error
		GetUpdateChannel(offset, limit, timeout int) (<-chan tgBot.Update, error)
		DownloadFile(fileID string) ([]byte, error)
	}

	client struct {
		bot *tgBot.BotAPI
	}
)

func NewTGBotClient(token string) (Client, error) {
	bot, err := tgBot.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("faield to create the bot: %w", err)
	}

	return &client{
		bot: bot,
	}, nil
}

func (c *client) Send(msg string, chatID int64) error {
	msgConfig := tgBot.NewMessage(chatID, msg)
	_, err := c.bot.Send(msgConfig)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) GetUpdateChannel(offset, limit, timeout int) (<-chan tgBot.Update, error) {
	return c.bot.GetUpdatesChan(c.newUpdateConfig(offset, limit, timeout))
}

func (c *client) DownloadFile(fileID string) ([]byte, error) {
	url, err := c.bot.GetFileDirectURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file direct url: %w", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to do get request: %w", err)
	}
	defer func() {
		if resp != nil {
			if errClose := resp.Body.Close(); errClose != nil {
				log.Print(errClose)
			}
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", resp.Status)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return content, nil
}

func (c *client) newUpdateConfig(offset, limit, timeout int) tgBot.UpdateConfig {
	updateConf := tgBot.NewUpdate(offset)
	updateConf.Limit = limit
	updateConf.Timeout = timeout

	return updateConf
}

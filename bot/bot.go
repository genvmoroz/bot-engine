package bot

import (
	"fmt"
	baseHTTP "net/http"

	"github.com/genvmoroz/client-go/http"
	base "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Client struct {
	Bot *base.BotAPI
}

func NewClient(token string) (*Client, error) {
	bot, err := base.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Client{
		Bot: bot,
	}, nil
}

func (c *Client) Send(chatID int64, msg string) error {
	msgConfig := base.NewMessage(chatID, msg)
	_, err := c.Bot.Send(msgConfig)
	return err
}

func (c *Client) SendWithParseMode(chatID int64, msg string, mode string) error {
	msgConfig := base.NewMessage(chatID, msg)
	msgConfig.ParseMode = mode
	_, err := c.Bot.Send(msgConfig)
	return err
}

type (
	Update         = base.Update
	UpdatesChannel = base.UpdatesChannel
)

func (c *Client) GetUpdateChannel(offset, limit, timeout int) UpdatesChannel {
	return c.Bot.GetUpdatesChan(c.newUpdateConfig(offset, limit, timeout))
}

func (c *Client) DownloadFile(fileID string) ([]byte, error) {
	url, err := c.Bot.GetFileDirectURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("get file direct url: %w", err)
	}

	client, err := http.NewClient()
	if err != nil {
		return nil, fmt.Errorf("create http client: %w", err)
	}

	req := http.AcquireRequest()
	defer http.ReleaseRequest(req)

	req.Header.SetRequestURI(url)
	req.Header.SetMethod(baseHTTP.MethodGet)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept-Charset", "utf-8")

	resp, err := client.Do(req)
	defer func() {
		if resp != nil {
			http.ReleaseResponse(resp)
		}
	}()
	switch {
	case err != nil:
		return nil, fmt.Errorf("execute GET request: %w", err)
	case resp == nil:
		return nil, fmt.Errorf("nullable response from the server, url: %s", url)
	case resp.StatusCode() != baseHTTP.StatusOK:
		return nil, fmt.Errorf("status: %d %s", resp.StatusCode(), baseHTTP.StatusText(resp.StatusCode()))
	}

	return resp.Body(), nil
}

func (c *Client) newUpdateConfig(offset, limit, timeout int) base.UpdateConfig {
	updateConf := base.NewUpdate(offset)
	updateConf.Limit = limit
	updateConf.Timeout = timeout

	return updateConf
}

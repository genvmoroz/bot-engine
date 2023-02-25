package bot

import (
	"fmt"
	baseHTTP "net/http"

	base "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/genvmoroz/client-go/http"
)

type Client struct {
	bot *base.BotAPI
}

func NewClient(token string) (*Client, error) {
	bot, err := base.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Client{
		bot: bot,
	}, nil
}

func (c *Client) Send(chatID int64, msg string) error {
	msgConfig := base.NewMessage(chatID, msg)
	_, err := c.bot.Send(msgConfig)
	return err
}

type (
	Update         = base.Update
	UpdatesChannel = base.UpdatesChannel
)

func (c *Client) GetUpdateChannel(offset, limit, timeout int) UpdatesChannel {
	return c.bot.GetUpdatesChan(c.newUpdateConfig(offset, limit, timeout))
}

func (c *Client) DownloadFile(fileID string) ([]byte, error) {
	url, err := c.bot.GetFileDirectURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get file direct url: %w", err)
	}

	client, err := http.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create http client: %w", err)
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
		return nil, fmt.Errorf("failed to execute get request: %w", err)
	case resp == nil:
		return nil, fmt.Errorf("nullable response from server, url: %s", url)
	case resp.StatusCode() != baseHTTP.StatusOK:
		return nil, fmt.Errorf("status: %s %d", baseHTTP.StatusText(resp.StatusCode()), resp.StatusCode())
	}

	return resp.Body(), nil
}

func (c *Client) newUpdateConfig(offset, limit, timeout int) base.UpdateConfig {
	updateConf := base.NewUpdate(offset)
	updateConf.Limit = limit
	updateConf.Timeout = timeout

	return updateConf
}

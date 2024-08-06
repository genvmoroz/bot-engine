package tg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	baseHTTP "net/http"

	base "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/samber/lo"
)

type (
	Doer interface {
		Send(c base.Chattable) (base.Message, error)
		GetUpdatesChan(config base.UpdateConfig) UpdatesChannel
		GetFileDirectURL(fileID string) (string, error)
	}

	Client struct {
		doer Doer
	}
)

func NewClient(token string) (*Client, error) {
	doer, err := base.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return newClient(doer), nil
}

func newClient(doer Doer) *Client {
	return &Client{doer: doer}
}

const msgLim = 4096

func (c *Client) Send(chatID int64, msg string) error {
	runes := []rune(msg)
	for _, chunk := range lo.Chunk(runes, msgLim) {
		_, err := c.doer.Send(base.NewMessage(chatID, string(chunk)))
		if err != nil {
			return err
		}
	}

	return nil
}

type ParseMode int

const (
	ModeMarkdown ParseMode = iota + 1
	ModeMarkdownV2
	ModeHTML
)

func (c *Client) SendWithParseMode(chatID int64, msg string, mode ParseMode) error {
	apiMode, err := transformParseModeToAPI(mode)
	if err != nil {
		return fmt.Errorf("transform parse mode to API: %w", err)
	}

	runes := []rune(msg)
	for _, chunk := range lo.Chunk(runes, msgLim) {
		msgConfig := base.NewMessage(chatID, string(chunk))
		msgConfig.ParseMode = apiMode
		_, err = c.doer.Send(msgConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) SendAudio(chatID int64, name string, bytes []byte) error {
	audio := base.NewAudio(chatID,
		base.FileBytes{
			Name:  name,
			Bytes: bytes,
		},
	)
	_, err := c.doer.Send(audio)
	return err
}

type (
	Update         = base.Update
	UpdatesChannel = base.UpdatesChannel
)

func (c *Client) GetUpdateChannel(offset, limit, timeout int) UpdatesChannel {
	cfg := base.NewUpdate(offset)
	cfg.Limit = limit
	cfg.Timeout = timeout

	return c.doer.GetUpdatesChan(cfg)
}

func (c *Client) DownloadFile(ctx context.Context, fileID string) ([]byte, error) {
	url, err := c.doer.GetFileDirectURL(fileID)
	if err != nil {
		return nil, fmt.Errorf("get file direct url: %w", err)
	}

	return c.doHTTPRequest(ctx, baseHTTP.MethodGet, url, nil)
}

func (c *Client) doHTTPRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	req, err := baseHTTP.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create a request: %w", err)
	}
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Accept-Charset", "utf-8")

	client := cleanhttp.DefaultPooledClient()
	resp, err := client.Do(req)
	switch {
	case err != nil:
		return nil, fmt.Errorf("do request: %w", err)
	case resp == nil:
		return nil, fmt.Errorf("nullable response from the server, url: %s", url)
	}

	defer func() {
		if resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != baseHTTP.StatusOK {
		return nil, fmt.Errorf(
			"status: %d [%s]. Body: %s",
			resp.StatusCode, baseHTTP.StatusText(resp.StatusCode), string(respBody),
		)
	}

	return respBody, nil
}

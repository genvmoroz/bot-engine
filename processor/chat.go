package processor

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/genvmoroz/bot-engine/tg"
	"github.com/sirupsen/logrus"
)

type (
	Client interface {
		Send(chatID int64, msg string) error
		SendWithParseMode(chatID int64, msg string, mode tg.ParseMode) error
		SendAudio(chatID int64, name string, bytes []byte) error
		DownloadFile(ctx context.Context, fileID string) ([]byte, error)
	}

	StateProcessor interface {
		Process(ctx context.Context, client Client, chatID int64, updateChan tg.UpdatesChannel) error
		Command() string
		Description() string
	}

	ChatProcessor struct {
		chatID     int64
		client     Client
		updateChan chan tg.Update
		states     map[string]StateProcessor
	}
)

func NewChatProcessor(chatID int64, client Client, states map[string]StateProcessor) (*ChatProcessor, error) {
	if client == nil {
		return nil, errors.New("client is missing")
	}
	if len(states) == 0 {
		return nil, errors.New("states is missing. create one at least")
	}
	return &ChatProcessor{
		chatID:     chatID,
		client:     client,
		updateChan: make(chan tg.Update, 1),
		states:     states,
	}, nil
}

func (p *ChatProcessor) PutUpdate(update tg.Update) error {
	if p.chatID != update.Message.Chat.ID {
		return fmt.Errorf(
			"the message was not delivered, original chatID %d does not match with message chatID %d",
			p.chatID, update.Message.Chat.ID,
		)
	}

	go func() {
		p.updateChan <- update
	}()

	return nil
}

func (p *ChatProcessor) Process(ctx context.Context, wg *sync.WaitGroup) {
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("context canceled, chat[id:%d] is closed", p.chatID)
			return
		case update, ok := <-p.updateChan:
			if !ok {
				logrus.Infof("updateChan is closed, chat[id:%d] is closed", p.chatID)
				return
			}
			if err := p.processUpdate(ctx, wg, update); err != nil {
				msg := fmt.Sprintf("failed to process an update for chat[id:%d]: %s", p.chatID, err.Error())
				p.sendMessage(msg)
				logrus.Error(msg)
			}
		}
	}
}

func (p *ChatProcessor) processUpdate(ctx context.Context, wg *sync.WaitGroup, update tg.Update) error {
	state := update.Message.Text
	stateProcessor, exist := p.states[state]
	if exist {
		wg.Add(1)
		defer wg.Done()

		inCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		if err := stateProcessor.Process(inCtx, p.client, p.chatID, p.updateChan); err != nil {
			return fmt.Errorf("process the state %s: %w", state, err)
		}
	} else {
		p.sendMessage("Unknown command")
	}

	p.sendMessage("You're in the main state")

	return nil
}

func (p *ChatProcessor) sendMessage(msg string) {
	if err := p.client.Send(p.chatID, msg); err != nil {
		logrus.Errorf("send the message [%s] to the chat [ID:%d]: %s", msg, p.chatID, err.Error())
	}
}

func (p *ChatProcessor) Close() {
	clear(p.states)
	close(p.updateChan)
}

package processor

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/genvmoroz/bot-engine/bot"
)

type ChatProcessor struct {
	chatID     int64
	tgBot      *bot.Client
	updateChan chan bot.Update
	states     map[string]StateProcessor
}

func NewChatProcessor(chatID int64, tgBot *bot.Client, states map[string]StateProcessor) (*ChatProcessor, error) {
	if tgBot == nil {
		return nil, errors.New("tgBot cannot be nil")
	}
	if len(states) == 0 {
		return nil, errors.New("states cannot be empty. create one at least")
	}
	return &ChatProcessor{
		chatID:     chatID,
		tgBot:      tgBot,
		updateChan: make(chan bot.Update, 1),
		states:     states,
	}, nil
}

func (p *ChatProcessor) PutUpdate(update bot.Update) error {
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
			p.processUpdate(ctx, wg, update)
		}
	}
}

func (p *ChatProcessor) processUpdate(ctx context.Context, wg *sync.WaitGroup, update bot.Update) {
	state := update.Message.Text
	stateProcessor, exist := p.states[state]
	if exist {
		wg.Add(1)
		if err := stateProcessor.Process(ctx, p.tgBot, p.chatID, p.updateChan); err != nil {
			logrus.Errorf("failed to process the state %s, chatID: %d, error: %s", state, p.chatID, err.Error())
		}
		wg.Done()
	} else {
		msg := "Unknown command"
		if err := p.tgBot.Send(p.chatID, msg); err != nil {
			logrus.Errorf("failed to send the message [%s] to the chat processor[ID:%d]: %s", msg, p.chatID, err.Error())
		}
	}
	msg := "You're in the main state"
	if err := p.tgBot.Send(p.chatID, msg); err != nil {
		logrus.Errorf("failed to send the message [%s] to the chat processor[ID:%d]: %s", msg, p.chatID, err.Error())
	}
}

func (p *ChatProcessor) GetChatID() int64 {
	if p == nil {
		return 0
	}
	return p.chatID
}

func (p *ChatProcessor) Close() {
	p.states = nil
	close(p.updateChan)
}

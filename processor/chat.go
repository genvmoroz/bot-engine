package processor

import (
	"context"
	"errors"
	"fmt"
	"sync"

	bot "github.com/genvmoroz/bot-engine/api"
	tgBotApi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type ChatProcessor struct {
	chatID            int64
	tgBot             bot.Client
	updateChan        chan tgBotApi.Update
	stateProcessorMap map[string]StateProcessor
}

func New(chatID int64, tgBot bot.Client, stateProcessorMap map[string]StateProcessor) (*ChatProcessor, error) {
	if tgBot == nil {
		return nil, errors.New("tgBot cannot be nil")
	}
	if stateProcessorMap == nil {
		return nil, errors.New("stateProcessorMap cannot be nil")
	}
	return &ChatProcessor{
		chatID:            chatID,
		tgBot:             tgBot,
		updateChan:        make(chan tgBotApi.Update, 1),
		stateProcessorMap: stateProcessorMap,
	}, nil
}

func (p *ChatProcessor) PutUpdate(update tgBotApi.Update) error {
	if update.Message.Chat.ID != p.chatID {
		return errors.New("the message was not delivered, chatIDs do not match")
	}

	go func() {
		p.updateChan <- update
	}()

	return nil
}

func (p *ChatProcessor) Process(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			msg := fmt.Sprintf("context canceled, chat[id:%d] is closed", p.chatID)
			sendAndPrint(msg, p.chatID, p.tgBot)
			return
		case update, ok := <-p.updateChan:
			if !ok {
				msg := fmt.Sprintf("updateChan is closed, chat[id:%d] is closed", p.chatID)
				sendAndPrint(msg, p.chatID, p.tgBot)
				return
			}
			p.processUpdate(ctx, wg, update)
		}
	}
}

func (p *ChatProcessor) processUpdate(ctx context.Context, wg *sync.WaitGroup, update tgBotApi.Update) {
	text := update.Message.Text
	stateProcessor, exist := p.stateProcessorMap[text]
	if exist {
		wg.Add(1)
		if err := stateProcessor.Process(ctx, wg, p.updateChan); err != nil {
			msg := fmt.Sprintf("failed to process the state %s, chatID: %d, error: %s", text, p.chatID, err.Error())
			sendAndPrint(msg, p.chatID, p.tgBot)
		}
	} else {
		msg := "Unknown command. You may choose current state by command, to see all available commands enter /help"
		if err := p.tgBot.Send(msg, p.chatID); err != nil {
			msg := fmt.Sprintf("failed to send the message to the chat processor[ID:%d]: %s", p.chatID, err.Error())
			sendAndPrint(msg, p.chatID, p.tgBot)
		}
	}
	if err := p.tgBot.Send("You're in the main state", p.chatID); err != nil {
		msg := fmt.Sprintf("failed to send the message to the chat processor[ID:%d]: %s", p.chatID, err.Error())
		sendAndPrint(msg, p.chatID, p.tgBot)
	}
}

func (p ChatProcessor) GetChatID() int64 {
	return p.chatID
}

func (p *ChatProcessor) Close() error { // nolint:unparam
	p.stateProcessorMap = nil
	close(p.updateChan)

	return nil
}

package dispatcher

import (
	"errors"
	"fmt"
	"log"

	bot "github.com/genvmoroz/bot-engine/api"
	"github.com/genvmoroz/bot-engine/processor"
	tgBotApi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Dispatcher struct {
	tgBot                       bot.Client
	chatProcessorMap            map[int64]*processor.ChatProcessor
	createStateProcessorMapFunc func(bot.Client, int64) map[string]processor.StateProcessor
}

func New(tgBot bot.Client, createStateProcessorMapFunc func(bot.Client, int64) map[string]processor.StateProcessor) (*Dispatcher, error) {
	if tgBot == nil {
		return nil, errors.New("tgBot cannot be nil")
	}
	if createStateProcessorMapFunc == nil {
		return nil, errors.New("createStateProcessorMapFunc cannot be nil")
	}

	return &Dispatcher{
		tgBot:                       tgBot,
		chatProcessorMap:            make(map[int64]*processor.ChatProcessor),
		createStateProcessorMapFunc: createStateProcessorMapFunc,
	}, nil
}

func (d *Dispatcher) Dispatch(updateChan <-chan tgBotApi.Update) error {
	if d == nil {
		return errors.New("dispatcher cannot be nil")
	}
	if updateChan == nil {
		return errors.New("updateChan cannot be nil")
	}

	for {
		update, ok := <-updateChan
		if !ok {
			log.Fatal("updateChan is closed")
		}

		chatID := update.Message.Chat.ID
		existedChatProcessor, exist := d.chatProcessorMap[chatID]
		if exist {
			go d.putUpdateIntoChatProcessorAndLog(existedChatProcessor, update)
		} else {
			newChatProcess, err := processor.New(chatID, d.tgBot, d.createStateProcessorMapFunc(d.tgBot, chatID))
			if err != nil {
				return fmt.Errorf("failed to create a new processor[ID:%d]: %w", chatID, err)
			}
			d.chatProcessorMap[chatID] = newChatProcess

			go newChatProcess.Process()
			go d.putUpdateIntoChatProcessorAndLog(newChatProcess, update)
		}
	}
}

func (d *Dispatcher) putUpdateIntoChatProcessorAndLog(p *processor.ChatProcessor, update tgBotApi.Update) {
	if err := p.PutUpdate(update); err != nil {
		log.Printf("failed to put the update into the processor[ID:%d]: %s", p.GetChatID(), err.Error())
	}
}

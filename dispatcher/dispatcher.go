package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/genvmoroz/bot-engine/bot"
	"github.com/genvmoroz/bot-engine/processor"
)

type (
	StateProvider func(*bot.Client, int64) map[string]processor.StateProcessor

	Dispatcher struct {
		tgBot            *bot.Client
		chatProcessorMap map[int64]*processor.ChatProcessor
		stateProvider    StateProvider

		timeout uint
	}
)

func New(tgBot *bot.Client, stateProvider StateProvider, timeout uint) (*Dispatcher, error) {
	if tgBot == nil {
		return nil, errors.New("tgBot cannot be nil")
	}
	if stateProvider == nil {
		return nil, errors.New("StateProvider cannot be nil")
	}

	return &Dispatcher{
		tgBot:            tgBot,
		chatProcessorMap: make(map[int64]*processor.ChatProcessor),
		stateProvider:    stateProvider,
		timeout:          timeout,
	}, nil
}

func (d *Dispatcher) Dispatch(ctx context.Context, wg *sync.WaitGroup, offset, limit int) error {
	if d == nil {
		return errors.New("dispatcher cannot be nil")
	}

	updateChan := d.tgBot.GetUpdateChannel(offset, limit, int(d.timeout))
	defer updateChan.Clear()

	for {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-updateChan:
			if !ok {
				return errors.New("updateChan is closed")
			}
			if err := d.dispatchUpdate(ctx, wg, update); err != nil {
				return fmt.Errorf("failed to dispatch update: %w", err)
			}
		}
	}
}

func (d *Dispatcher) dispatchUpdate(ctx context.Context, wg *sync.WaitGroup, update bot.Update) error {
	chatID := update.Message.Chat.ID

	if exist := d.putUpdateIntoExistedChatProcessor(chatID, update); !exist {
		if err := d.createChatProcessor(ctx, wg, chatID); err != nil {
			return fmt.Errorf("failed to create ChatProcessor [ID:%d]: %w", chatID, err)
		}
		if ok := d.putUpdateIntoExistedChatProcessor(chatID, update); !ok {
			return fmt.Errorf("unexpected error: no chat processor created with ID: %d", chatID)
		}
	}

	return nil
}

func (d *Dispatcher) putUpdateIntoExistedChatProcessor(chatID int64, update bot.Update) bool {
	existedChatProcessor, exist := d.chatProcessorMap[chatID]
	if !exist {
		return false
	}

	if err := existedChatProcessor.PutUpdate(update); err != nil {
		logrus.Errorf(
			"failed to put the update into the chat [ID:%d]: %s",
			existedChatProcessor.GetChatID(), err.Error(),
		)
	}
	return true
}

func (d *Dispatcher) createChatProcessor(ctx context.Context, wg *sync.WaitGroup, chatID int64) error {
	newChatProcessor, err := processor.New(chatID, d.tgBot, d.stateProvider(d.tgBot, chatID))
	if err != nil {
		return err
	}
	d.chatProcessorMap[chatID] = newChatProcessor

	wg.Add(1)
	go func() {
		defer wg.Done()
		newChatProcessor.Process(ctx, wg)
	}()

	return nil
}

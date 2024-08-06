package dispatcher

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/genvmoroz/bot-engine/processor"
	"github.com/genvmoroz/bot-engine/tg"
	"github.com/sirupsen/logrus"
)

type (
	Client interface {
		Send(chatID int64, msg string) error
		GetUpdateChannel(offset, limit, timeout int) tg.UpdatesChannel
	}

	States map[string]processor.StateProcessor

	Dispatcher struct {
		client           Client
		chatProcessorMap map[int64]*processor.ChatProcessor
		states           States
		timeout          uint
	}
)

func New(client Client, states States, timeout uint) (*Dispatcher, error) {
	if client == nil {
		return nil, errors.New("client is missing")
	}
	if len(states) == 0 {
		return nil, errors.New("states is missing, create one at least")
	}

	return &Dispatcher{
		client:           client,
		chatProcessorMap: make(map[int64]*processor.ChatProcessor),
		states:           states,
		timeout:          timeout,
	}, nil
}

func (d *Dispatcher) Dispatch(ctx context.Context, wg *sync.WaitGroup, offset, limit int) error {
	updateChan := d.client.GetUpdateChannel(offset, limit, int(d.timeout))
	defer updateChan.Clear()

	for {
		select {
		case <-ctx.Done():
			logrus.Info("context canceled, dispatcher is closed")
			return nil
		case update, ok := <-updateChan:
			if !ok {
				return errors.New("updateChan is closed")
			}
			if err := d.dispatchUpdate(ctx, wg, update); err != nil {
				return fmt.Errorf("dispatch update: %w", err)
			}
		}
	}
}

func (d *Dispatcher) dispatchUpdate(ctx context.Context, wg *sync.WaitGroup, update tg.Update) error {
	chatID := update.Message.Chat.ID

	existedChatProcessor, exist := d.chatProcessorMap[chatID]
	if !exist {
		if err := d.createChatProcessor(ctx, wg, chatID); err != nil {
			return fmt.Errorf("create ChatProcessor [ID:%d]: %w", chatID, err)
		}
	}

	if err := existedChatProcessor.PutUpdate(update); err != nil {
		return fmt.Errorf("put the update into the chat [ID:%d]: %w", chatID, err)
	}

	return nil
}

func (d *Dispatcher) createChatProcessor(ctx context.Context, wg *sync.WaitGroup, chatID int64) error {
	newChatProcessor, err := processor.NewChatProcessor(chatID, d.client, d.states)
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

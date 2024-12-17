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
		SendWithParseMode(chatID int64, msg string, mode tg.ParseMode) error
		SendAudio(chatID int64, name string, bytes []byte) error
		DownloadFile(ctx context.Context, fileID string) ([]byte, error)
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
	if update.Message == nil || update.Message.Chat == nil {
		logrus.Infof("message or chat is missing: %v", update)
		return nil
	}

	chatID := update.Message.Chat.ID

	chatProcessor, exist := d.chatProcessorMap[chatID]
	if !exist {
		var err error
		chatProcessor, err = d.createChatProcessor(ctx, wg, chatID)
		if err != nil {
			return fmt.Errorf("create ChatProcessor [ID:%d]: %w", chatID, err)
		}
	}

	if err := chatProcessor.PutUpdate(update); err != nil {
		return fmt.Errorf("put the update into the chat [ID:%d]: %w", chatID, err)
	}

	return nil
}

func (d *Dispatcher) createChatProcessor(
	ctx context.Context, wg *sync.WaitGroup, chatID int64,
) (*processor.ChatProcessor, error) {
	newChatProcessor, err := processor.NewChatProcessor(chatID, d.client, d.states)
	if err != nil {
		return nil, err
	}
	d.chatProcessorMap[chatID] = newChatProcessor

	wg.Add(1)
	go func() {
		defer wg.Done()
		newChatProcessor.Process(ctx, wg)
	}()

	return newChatProcessor, nil
}

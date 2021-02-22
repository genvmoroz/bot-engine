package processor

import (
	"context"
	"sync"

	tgBotApi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type StateProcessor interface {
	Process(context.Context, *sync.WaitGroup, <-chan tgBotApi.Update) error
}

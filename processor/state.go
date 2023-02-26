package processor

import (
	"context"

	"github.com/genvmoroz/bot-engine/bot"
)

type StateProcessor interface {
	Process(ctx context.Context, client *bot.Client, chatID int64, updateChan bot.UpdatesChannel) error
	Command() string
	Description() string
}

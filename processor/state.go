package processor

import (
	"context"

	"github.com/genvmoroz/bot-engine/bot"
)

type StateProcessor interface {
	Process(context.Context, bot.UpdatesChannel) error
	Command() string
	Description() string
}

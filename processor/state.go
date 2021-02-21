package processor

import (
	tgBotApi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type StateProcessor interface {
	Process(<-chan tgBotApi.Update) error
}

package processor

import (
	tgBotApi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type StateProcessor interface {
	Process(channel tgBotApi.UpdatesChannel) error
}

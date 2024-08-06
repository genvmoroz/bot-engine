package tg

import (
	"fmt"

	base "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func transformParseModeToAPI(in ParseMode) (string, error) {
	switch in {
	case ModeMarkdown:
		return base.ModeMarkdown, nil
	case ModeMarkdownV2:
		return base.ModeMarkdownV2, nil
	case ModeHTML:
		return base.ModeHTML, nil
	default:
		return "", fmt.Errorf("unknown parse mode: %d", in)
	}
}

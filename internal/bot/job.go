package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/jrdev95/super-picos-downloader/internal/platform"
)

const downloadQueueSize = 100

type downloadJob struct {
	message  *tgbotapi.Message
	url      string
	platform platform.Platform
}

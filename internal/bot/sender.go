package bot

import (
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/jrdev95/super-picos-downloader/internal/media"
)

const maxMediaGroupSize = 10

func (b *Bot) sendResult(
	message *tgbotapi.Message,
	result *media.Result,
) error {
	if message == nil {
		return errors.New("mensagem de origem não definida")
	}

	if result == nil {
		return errors.New("resultado de mídia não definido")
	}

	if err := result.Validate(); err != nil {
		return fmt.Errorf(
			"resultado de mídia inválido: %w",
			err,
		)
	}

	if len(result.Items) == 1 {
		return b.sendSingle(
			message.Chat.ID,
			message.MessageID,
			result.Items[0],
		)
	}

	batches := splitMediaBatches(result.Items)

	for index, batch := range batches {
		replyTo := 0

		// Apenas o primeiro envio responde diretamente
		// à mensagem original.
		if index == 0 {
			replyTo = message.MessageID
		}

		if len(batch) == 1 {
			if err := b.sendSingle(
				message.Chat.ID,
				replyTo,
				batch[0],
			); err != nil {
				return err
			}

			continue
		}

		if err := b.sendAlbum(
			message.Chat.ID,
			replyTo,
			batch,
		); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bot) sendSingle(
	chatID int64,
	replyTo int,
	item media.Item,
) error {
	file := tgbotapi.FilePath(item.Path)

	switch item.Type {
	case media.Photo:
		config := tgbotapi.NewPhoto(
			chatID,
			file,
		)

		config.Caption = item.Caption
		config.ReplyToMessageID = replyTo

		if _, err := b.api.Send(config); err != nil {
			return fmt.Errorf(
				"falha ao enviar foto: %w",
				err,
			)
		}

	case media.Video:
		config := tgbotapi.NewVideo(
			chatID,
			file,
		)

		config.Caption = item.Caption
		config.ReplyToMessageID = replyTo

		if _, err := b.api.Send(config); err != nil {
			return fmt.Errorf(
				"falha ao enviar vídeo: %w",
				err,
			)
		}

	default:
		return fmt.Errorf(
			"tipo de mídia não suportado: %q",
			item.Type,
		)
	}

	return nil
}

func (b *Bot) sendAlbum(
	chatID int64,
	replyTo int,
	items []media.Item,
) error {
	if len(items) < 2 {
		return errors.New(
			"álbum precisa conter pelo menos 2 mídias",
		)
	}

	if len(items) > maxMediaGroupSize {
		return fmt.Errorf(
			"álbum excede limite de %d mídias",
			maxMediaGroupSize,
		)
	}

	inputMedia := make(
		[]interface{},
		0,
		len(items),
	)

	for _, item := range items {
		file := tgbotapi.FilePath(item.Path)

		switch item.Type {
		case media.Photo:
			photo := tgbotapi.NewInputMediaPhoto(file)
			photo.Caption = item.Caption

			inputMedia = append(
				inputMedia,
				photo,
			)

		case media.Video:
			video := tgbotapi.NewInputMediaVideo(file)
			video.Caption = item.Caption

			inputMedia = append(
				inputMedia,
				video,
			)

		default:
			return fmt.Errorf(
				"tipo de mídia não suportado no álbum: %q",
				item.Type,
			)
		}
	}

	group := tgbotapi.NewMediaGroup(
		chatID,
		inputMedia,
	)

	group.ReplyToMessageID = replyTo

	if _, err := b.api.SendMediaGroup(group); err != nil {
		return fmt.Errorf(
			"falha ao enviar álbum: %w",
			err,
		)
	}

	return nil
}

func splitMediaBatches(
	items []media.Item,
) [][]media.Item {
	if len(items) == 0 {
		return nil
	}

	var batches [][]media.Item

	for len(items) > 0 {
		size := maxMediaGroupSize

		if len(items) < size {
			size = len(items)
		}

		// Evita terminar com apenas 1 mídia,
		// já que MediaGroup exige pelo menos 2.
		if len(items) == maxMediaGroupSize+1 {
			size = maxMediaGroupSize - 1
		}

		batch := make(
			[]media.Item,
			size,
		)

		copy(batch, items[:size])

		batches = append(
			batches,
			batch,
		)

		items = items[size:]
	}

	return batches
}

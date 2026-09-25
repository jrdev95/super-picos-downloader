package bot

import (
	"context"
	"log/slog"
)

func (b *Bot) startWorkers(
	ctx context.Context,
) {
	for workerID := 1; workerID <= b.workers; workerID++ {
		b.wg.Add(1)

		go b.worker(
			ctx,
			workerID,
		)
	}

	slog.Info(
		"workers iniciados",
		"workers", b.workers,
		"queue_capacity", cap(b.jobs),
	)
}

func (b *Bot) worker(
	ctx context.Context,
	workerID int,
) {
	defer b.wg.Done()

	slog.Info(
		"worker iniciado",
		"worker_id", workerID,
	)

	for {
		select {
		case <-ctx.Done():
			slog.Info(
				"worker encerrado",
				"worker_id", workerID,
			)

			return

		case job := <-b.jobs:
			slog.Info(
				"worker iniciou download",
				"worker_id", workerID,
				"platform", job.platform,
				"url", job.url,
			)

			b.processDownload(
				ctx,
				job.message,
				job.url,
				job.platform,
			)

			slog.Info(
				"worker finalizou download",
				"worker_id", workerID,
			)
		}
	}
}

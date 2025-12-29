package initiator

import (
	"context"
	"log/slog"
	"time"

	"github.com/liy0aay/xkcd-search/search/core"
)

func RunIndexUpdate(
	ctx context.Context, searcher core.Searcher, period time.Duration, log *slog.Logger,
) {
	go func() {
		// update on start
		if err := searcher.BuildIndex(ctx); err != nil {
			log.Error("failed to build index on start", "error", err)
		}
		ticker := time.NewTicker(period)
		for {
			select {
			case <-ctx.Done():
				log.Error("quit updater")
			case <-ticker.C:
				log.Info("run index update")
				if err := searcher.BuildIndex(ctx); err != nil {
					log.Error("build index failed", "error", err)
				}
			}
		}
	}()
}

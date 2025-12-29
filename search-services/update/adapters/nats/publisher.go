package nats

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/liy0aay/xkcd-search/events"
	"github.com/liy0aay/xkcd-search/update/core"
	natslib "github.com/nats-io/nats.go"
)

var _ core.Publisher = (*Publisher)(nil)

type Publisher struct {
	nc  *natslib.Conn
	log *slog.Logger
}

func New(log *slog.Logger, brokerAddress string) (*Publisher, error) {
	opts := []natslib.Option{
		natslib.Name("update-service"),
		natslib.ReconnectHandler(func(_ *natslib.Conn) {
			log.Info("NATS reconnected")
		}),
		natslib.DisconnectErrHandler(func(_ *natslib.Conn, err error) {
			if err != nil {
				log.Warn("NATS disconnected", "error", err)
			} else {
				log.Warn("NATS disconnected")
			}
		}),
		natslib.ErrorHandler(func(_ *natslib.Conn, _ *natslib.Subscription, err error) {
			log.Error("NATS error", "error", err)
		}),
	}

	nc, err := natslib.Connect(brokerAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to broker: %v", err)
	}

	return &Publisher{nc: nc, log: log}, nil
}

func (p *Publisher) PublishDBUpdateEvent(ctx context.Context) error {
	p.log.Info("publishing event: db updated")
	if err := p.nc.Publish(events.TopicDBUpdated, []byte("updated")); err != nil {
		p.log.Error("failed to publish db update event", "error", err)
		return fmt.Errorf("failed to publish db update event: %v", err)
	}
	if err := p.nc.Flush(); err != nil {
		p.log.Error("failed to flush db update event", "error", err)
		return fmt.Errorf("failed to flush db update event: %v", err)
	}
	return nil
}

func (p *Publisher) PublishDBDropEvent(ctx context.Context) error {
	p.log.Info("publishing event: db dropped")
	if err := p.nc.Publish(events.TopicDBDropped, []byte("dropped")); err != nil {
		p.log.Error("failed to publish db drop event", "error", err)
		return fmt.Errorf("failed to publish db drop event: %v", err)
	}
	if err := p.nc.Flush(); err != nil {
		p.log.Error("failed to flush db drop event", "error", err)
		return fmt.Errorf("failed to flush db drop event: %v", err)
	}
	return nil
}

func (p *Publisher) Close() error {
	if p.nc != nil {
		p.nc.Close()
	}
	return nil
}

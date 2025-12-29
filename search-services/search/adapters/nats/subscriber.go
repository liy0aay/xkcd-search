package nats

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/liy0aay/xkcd-search/events"
	natslib "github.com/nats-io/nats.go"
)

type Subscriber struct {
	nc   *natslib.Conn
	log  *slog.Logger
	subs []*natslib.Subscription
	mu   sync.Mutex
}

func New(log *slog.Logger, brokerAddress string) (*Subscriber, error) {
	opts := []natslib.Option{
		natslib.Name("search-service"),
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

	return &Subscriber{nc: nc, log: log}, nil
}

func (s *Subscriber) SubscribeDBUpdateEvent(ctx context.Context) (<-chan struct{}, error) {
	msgCh := make(chan *natslib.Msg, 10)
	sub, err := s.nc.ChanSubscribe(events.TopicDBUpdated, msgCh)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %v", events.TopicDBUpdated, err)
	}

	s.mu.Lock()
	s.subs = append(s.subs, sub)
	s.mu.Unlock()

	outCh := make(chan struct{})
	go func() {
		defer close(outCh)
		defer func() {
			if err := sub.Unsubscribe(); err != nil {
				s.log.Error("failed to unsubscribe from db update event", "error", err)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				s.log.Debug("stopping db update event listener")
				return
			case msg := <-msgCh:
				if msg == nil {
					return
				}
				s.log.Debug("received db update event", "data", string(msg.Data))
				outCh <- struct{}{}
			}
		}
	}()

	return outCh, nil
}

func (s *Subscriber) SubscribeDBDropEvent(ctx context.Context) (<-chan struct{}, error) {
	msgCh := make(chan *natslib.Msg, 10)
	sub, err := s.nc.ChanSubscribe(events.TopicDBDropped, msgCh)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to %s: %v", events.TopicDBDropped, err)
	}

	s.mu.Lock()
	s.subs = append(s.subs, sub)
	s.mu.Unlock()

	outCh := make(chan struct{})
	go func() {
		defer close(outCh)
		defer func() {
			if err := sub.Unsubscribe(); err != nil {
				s.log.Error("failed to unsubscribe from db drop event", "error", err)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				s.log.Debug("stopping db drop event listener")
				return
			case msg := <-msgCh:
				if msg == nil {
					return
				}
				s.log.Debug("received db drop event")
				outCh <- struct{}{}
			}
		}
	}()

	return outCh, nil
}

func (s *Subscriber) RunEventHandlers(ctx context.Context, updateHandler func(), dropHandler func()) error {
	updateCh, err := s.SubscribeDBUpdateEvent(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to db update events: %v", err)
	}

	dropCh, err := s.SubscribeDBDropEvent(ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to db drop events: %v", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				s.log.Debug("stopping event listener")
				return
			case <-updateCh:
				s.log.Info("handling db update event")
				updateHandler()
			case <-dropCh:
				s.log.Info("handling db drop event")
				dropHandler()
			}
		}
	}()

	return nil
}

	func (s *Subscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sub := range s.subs {
		if sub != nil {
			if err := sub.Unsubscribe(); err != nil {
				s.log.Error("failed to unsubscribe", "error", err)
			}
		}
	}
	s.subs = nil

	if s.nc != nil {
		s.nc.Close()
	}	return nil
}

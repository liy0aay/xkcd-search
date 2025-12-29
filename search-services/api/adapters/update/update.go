package update

import (
	"context"
	"errors"
	"log/slog"

	"github.com/liy0aay/xkcd-search/api/core"
	updatepb "github.com/liy0aay/xkcd-search/proto/update"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	log    *slog.Logger
	client updatepb.UpdateClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: updatepb.NewUpdateClient(conn),
		log:    log,
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, nil)
	return err
}

func (c *Client) Status(ctx context.Context) (core.UpdateStatus, error) {
	reply, err := c.client.Status(ctx, nil)
	if err != nil {
		return core.StatusUpdateUnknown, err
	}
	switch reply.Status {
	case updatepb.Status_STATUS_IDLE:
		return core.StatusUpdateIdle, nil
	case updatepb.Status_STATUS_RUNNING:
		return core.StatusUpdateRunning, nil
	}
	return core.StatusUpdateUnknown, errors.New("unknown status")
}

func (c *Client) Stats(ctx context.Context) (core.UpdateStats, error) {
	reply, err := c.client.Stats(ctx, nil)
	if err != nil {
		return core.UpdateStats{}, err
	}
	return core.UpdateStats{
		WordsTotal:    int(reply.GetWordsTotal()),
		WordsUnique:   int(reply.GetWordsUnique()),
		ComicsFetched: int(reply.GetComicsFetched()),
		ComicsTotal:   int(reply.GetComicsTotal()),
	}, nil
}

func (c *Client) Update(ctx context.Context) error {
	_, err := c.client.Update(ctx, nil)
	if status.Code(err) == codes.AlreadyExists {
		return core.ErrAlreadyExists
	}
	return err
}

func (c *Client) Drop(ctx context.Context) error {
	_, err := c.client.Drop(ctx, nil)
	return err
}

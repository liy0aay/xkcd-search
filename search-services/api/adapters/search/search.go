package search

import (
	"context"
	"log/slog"

	"github.com/liy0aay/xkcd-search/api/core"
	searchpb "github.com/liy0aay/xkcd-search/proto/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	log    *slog.Logger
	client searchpb.SearchClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{
		client: searchpb.NewSearchClient(conn),
		log:    log,
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	reply, err := c.client.Search(ctx, &searchpb.SearchRequest{
		Phrase: phrase, Limit: int64(limit),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	comics := make([]core.Comics, 0, len(reply.Comics))
	for _, c := range reply.Comics {
		comics = append(comics, core.Comics{ID: int(c.Id), URL: c.Url, Title: c.Title, Alt: c.Alt, Score: int(c.Score)})
	}
	return comics, nil
}

func (c *Client) SearchIndex(ctx context.Context, phrase string, limit int) ([]core.Comics, error) {
	reply, err := c.client.SearchIndex(ctx, &searchpb.SearchRequest{
		Phrase: phrase, Limit: int64(limit),
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, core.ErrNotFound
		}
		return nil, err
	}
	comics := make([]core.Comics, 0, len(reply.Comics))
	for _, c := range reply.Comics {
		comics = append(comics, core.Comics{ID: int(c.Id), URL: c.Url, Title: c.Title, Alt: c.Alt, Score: int(c.Score)})
	}
	return comics, nil
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, nil)
	return err
}

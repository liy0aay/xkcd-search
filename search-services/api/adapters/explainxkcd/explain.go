package explainxkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/liy0aay/xkcd-search/api/core"
	"github.com/liy0aay/xkcd-search/closers"
)

type Client struct {
	client http.Client
	url    string
	log    *slog.Logger
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{},
		url:    url,
		log:    log,
	}, nil
}

func (c *Client) Close() error {
	return nil
}

func (c Client) Explain(ctx context.Context, id int) (core.ExplainXKCDInfo, error) {
	reqURL := fmt.Sprintf(
		"%s/wiki/api.php?action=parse&page=%d&prop=text&sectiontitle=Explanation&redirects=1&format=json",
		c.url,
		id,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return core.ExplainXKCDInfo{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return core.ExplainXKCDInfo{}, err
	}
	defer closers.CloseOrLog(resp.Body, c.log)

	if resp.StatusCode == http.StatusNotFound {
		return core.ExplainXKCDInfo{}, core.ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return core.ExplainXKCDInfo{}, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var parsed struct {
		Parse struct {
			Text map[string]string `json:"text"`
		} `json:"parse"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return core.ExplainXKCDInfo{}, err
	}

	html, ok := parsed.Parse.Text["*"]
	if !ok {
		return core.ExplainXKCDInfo{}, fmt.Errorf("no explanation found")
	}

	return core.ExplainXKCDInfo{ID: id, HTML: html}, nil
}

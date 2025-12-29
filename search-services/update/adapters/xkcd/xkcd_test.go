package xkcd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/liy0aay/xkcd-search/update/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func testClient(rt http.RoundTripper) Client {
	return Client{
		client: http.Client{
			Transport: rt,
			Timeout:   time.Second,
		},
		url: "https://xkcd.com",
		log: slog.Default(),
	}
}

func TestGet_HappyPath(t *testing.T) {
	body := `{
		"num": 10,
		"img": "https://imgs.xkcd.com/comics/test.png",
		"title": "Title",
		"safe_title": "Safe",
		"transcript": "Transcript",
		"alt": "Alt"
	}`

	c := testClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}))

	info, err := c.Get(context.Background(), 10)
	require.NoError(t, err)

	assert.Equal(t, 10, info.ID)
	assert.Equal(t, "https://imgs.xkcd.com/comics/test.png", info.URL)
	assert.Contains(t, info.Description, "Title")
	assert.Contains(t, info.Description, "Safe")
}

func TestLastID(t *testing.T) {
	body := `{"num": 314}`

	c := testClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}))

	id, err := c.LastID(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 314, id)
}

func TestGet_NotFound(t *testing.T) {
	c := testClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}))

	_, err := c.Get(context.Background(), 1)
	require.Error(t, err)
	assert.True(t, errors.Is(err, core.ErrNotFound))
}

func TestGet_Timeout(t *testing.T) {
	c := testClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("timeout")
	}))

	_, err := c.Get(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to request comics")
}

func TestGet_InvalidJSON(t *testing.T) {
	c := testClient(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("{invalid json}")),
		}, nil
	}))

	_, err := c.Get(context.Background(), 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode comics")
}

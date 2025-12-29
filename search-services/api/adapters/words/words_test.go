package words

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/liy0aay/xkcd-search/api/core"
	wordspb "github.com/liy0aay/xkcd-search/proto/words"
)

type fakeWordsClient struct {
	normFunc func(ctx context.Context, req *wordspb.WordsRequest) (*wordspb.WordsReply, error)
	pingFunc func(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error)
}

func (f *fakeWordsClient) Norm(
	ctx context.Context,
	req *wordspb.WordsRequest,
	_ ...grpc.CallOption,
) (*wordspb.WordsReply, error) {
	return f.normFunc(ctx, req)
}

func (f *fakeWordsClient) Ping(
	ctx context.Context,
	req *emptypb.Empty,
	_ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	return f.pingFunc(ctx, req)
}

func newTestClient(fake *fakeWordsClient) *Client {
	return &Client{
		client: fake,
		log:    slog.Default(),
		conn:   &grpc.ClientConn{},
	}
}

func TestClient_Norm_ResourceExhausted(t *testing.T) {
	t.Parallel()

	fake := &fakeWordsClient{
		normFunc: func(ctx context.Context, req *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
			return nil, status.Error(codes.ResourceExhausted, "limit exceeded")
		},
	}

	client := newTestClient(fake)

	words, err := client.Norm(context.Background(), "test")

	require.Nil(t, words)
	require.ErrorIs(t, err, core.ErrBadArguments)
}

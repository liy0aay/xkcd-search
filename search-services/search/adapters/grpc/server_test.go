//go:generate mockgen -source=../../core/ports.go -destination=../../core/mocks/core_mocks.go -package=mocks
package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	searchpb "github.com/liy0aay/xkcd-search/proto/search"
	"github.com/liy0aay/xkcd-search/search/core"
	"github.com/liy0aay/xkcd-search/search/core/mocks"
)

func TestSearch_NotFoundMappedToGRPC(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockSearcher(ctrl)
	server := NewServer(mockSvc)

	mockSvc.EXPECT().
		Search(gomock.Any(), "abc", 10).
		Return(nil, core.ErrNotFound)

	_, err := server.Search(context.Background(), &searchpb.SearchRequest{
		Phrase: "abc",
		Limit:  10,
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestSearch_UnexpectedErrorPassedThrough(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSvc := mocks.NewMockSearcher(ctrl)
	server := NewServer(mockSvc)

	expectedErr := errors.New("boom")

	mockSvc.EXPECT().
		Search(gomock.Any(), "test", 10).
		Return(nil, expectedErr)

	_, err := server.Search(context.Background(), &searchpb.SearchRequest{
		Phrase: "test",
		Limit:  10,
	})

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

//go:generate mockgen -source=../../core/ports.go -destination=mocks_test.go -package=grpc
package grpc

import (
	"context"
	"errors"
	"testing"

	updatepb "github.com/liy0aay/xkcd-search/proto/update"
	"github.com/liy0aay/xkcd-search/update/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestStatus_Idle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	updater.EXPECT().
		Status(gomock.Any()).
		Return(core.StatusIdle)

	s := NewServer(updater, nil)

	resp, err := s.Status(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, updatepb.Status_STATUS_IDLE, resp.Status)
}

func TestStatus_Running(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	updater.EXPECT().
		Status(gomock.Any()).
		Return(core.StatusRunning)

	s := NewServer(updater, nil)

	resp, err := s.Status(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, updatepb.Status_STATUS_RUNNING, resp.Status)
}

func TestUpdate_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	publisher := NewMockPublisher(ctrl)

	updater.EXPECT().
		Update(gomock.Any()).
		Return(nil)

	publisher.EXPECT().
		PublishDBUpdateEvent(gomock.Any()).
		Return(nil)

	s := NewServer(updater, publisher)

	_, err := s.Update(context.Background(), nil)
	require.NoError(t, err)
}

func TestUpdate_AlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)

	updater.EXPECT().
		Update(gomock.Any()).
		Return(core.ErrAlreadyExists)

	s := NewServer(updater, nil)

	_, err := s.Update(context.Background(), nil)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
}

func TestUpdate_UnexpectedErrorPassedThrough(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	expectedErr := errors.New("boom")

	updater.EXPECT().
		Update(gomock.Any()).
		Return(expectedErr)

	s := NewServer(updater, nil)

	_, err := s.Update(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestUpdate_PublisherError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	publisher := NewMockPublisher(ctrl)

	updater.EXPECT().
		Update(gomock.Any()).
		Return(nil)

	publisher.EXPECT().
		PublishDBUpdateEvent(gomock.Any()).
		Return(errors.New("nats down"))

	s := NewServer(updater, publisher)

	_, err := s.Update(context.Background(), nil)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestStats_ErrorPassedThrough(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	expectedErr := errors.New("stats error")

	updater.EXPECT().
		Stats(gomock.Any()).
		Return(core.ServiceStats{}, expectedErr)

	s := NewServer(updater, nil)

	_, err := s.Stats(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestDrop_HappyPath(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	publisher := NewMockPublisher(ctrl)

	updater.EXPECT().
		Drop(gomock.Any()).
		Return(nil)

	publisher.EXPECT().
		PublishDBDropEvent(gomock.Any()).
		Return(nil)

	s := NewServer(updater, publisher)

	_, err := s.Drop(context.Background(), nil)
	require.NoError(t, err)
}

func TestDrop_ServiceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	expectedErr := errors.New("drop failed")

	updater.EXPECT().
		Drop(gomock.Any()).
		Return(expectedErr)

	s := NewServer(updater, nil)

	_, err := s.Drop(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
}

func TestDrop_PublisherError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	updater := NewMockUpdater(ctrl)
	publisher := NewMockPublisher(ctrl)

	updater.EXPECT().
		Drop(gomock.Any()).
		Return(nil)

	publisher.EXPECT().
		PublishDBDropEvent(gomock.Any()).
		Return(errors.New("nats error"))

	s := NewServer(updater, publisher)

	_, err := s.Drop(context.Background(), nil)
	require.Error(t, err)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

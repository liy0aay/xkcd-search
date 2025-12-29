package grpc

import (
	"context"
	"errors"

	updatepb "github.com/liy0aay/xkcd-search/proto/update"
	"github.com/liy0aay/xkcd-search/update/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewServer(service core.Updater, publisher core.Publisher) *Server {
	return &Server{service: service, publisher: publisher}
}

type Server struct {
	updatepb.UnimplementedUpdateServer
	service   core.Updater
	publisher core.Publisher
}

func (s *Server) Ping(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *Server) Status(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatusReply, error) {
	st := s.service.Status(ctx)

	switch st {
	case core.StatusIdle:
		return &updatepb.StatusReply{Status: updatepb.Status_STATUS_IDLE}, nil
	case core.StatusRunning:
		return &updatepb.StatusReply{Status: updatepb.Status_STATUS_RUNNING}, nil
	}
	return nil, status.Error(codes.Internal, "unknown status from service")
}

func (s *Server) Update(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Update(ctx); err != nil {
		if errors.Is(err, core.ErrAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "update already runs")
		}
		return nil, err
	}
	if err := s.publisher.PublishDBUpdateEvent(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return nil, nil
}

func (s *Server) Stats(ctx context.Context, _ *emptypb.Empty) (*updatepb.StatsReply, error) {
	stats, err := s.service.Stats(ctx)
	if err != nil {
		return nil, err
	}

	return &updatepb.StatsReply{
		WordsTotal:    int64(stats.DBStats.WordsTotal),
		WordsUnique:   int64(stats.DBStats.WordsUnique),
		ComicsTotal:   int64(stats.ComicsTotal),
		ComicsFetched: int64(stats.DBStats.ComicsFetched),
	}, nil
}

func (s *Server) Drop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Drop(ctx); err != nil {
		return nil, err
	}
	if err := s.publisher.PublishDBDropEvent(ctx); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return nil, nil
}

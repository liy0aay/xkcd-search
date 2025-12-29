package core

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"
)

type Service struct {
	log   *slog.Logger
	db    DB
	words Words
	index *Index
}

func NewService(log *slog.Logger, db DB, words Words) (*Service, error) {

	return &Service{
		log:   log,
		db:    db,
		words: words,
		index: NewIndex(),
	}, nil
}

func (s *Service) Search(ctx context.Context, phrase string, limit int) ([]Comics, error) {

	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("failed to find keywords", "error", err)
		return nil, err
	}
	s.log.Debug("normalized query", "keywords", keywords)

	// comics ID -> number of findings
	scores := map[int]int{}
	for _, keyword := range keywords {
		IDs, err := s.db.Search(ctx, keyword)
		if err != nil {
			s.log.Error("failed to search keyword in DB", "error", err)
			return nil, err
		}
		for _, ID := range IDs {
			scores[ID]++
		}
	}

	return s.fetch(ctx, scores, limit)
}

func (s *Service) SearchIndex(ctx context.Context, phrase string, limit int) ([]Comics, error) {

	keywords, err := s.words.Norm(ctx, phrase)
	if err != nil {
		s.log.Error("failed to find keywords", "error", err)
		return nil, err
	}
	s.log.Debug("normalized query", "keywords", keywords)

	// comics ID -> number of findings
	scores := map[int]int{}
	for _, keyword := range keywords {

		for _, ID := range s.index.Get(keyword) {
			scores[ID]++
		}
	}

	return s.fetch(ctx, scores, limit)
}

func (s *Service) fetch(ctx context.Context, scores map[int]int, limit int) ([]Comics, error) {
	s.log.Debug("relevant comics", "count", len(scores))

	// sort by number of findings
	sorted := slices.SortedFunc(maps.Keys(scores), func(a, b int) int {
		return cmp.Compare(scores[b], scores[a]) // desc
	})

	// limit results
	if len(sorted) < limit {
		limit = len(sorted)
	}
	sorted = sorted[:limit]

	// fetch comics
	result := make([]Comics, 0, len(sorted))
	for _, ID := range sorted {
		comics, err := s.db.Get(ctx, ID)
		if err != nil {
			s.log.Error("failed to fetch comics", "id", ID, "error", err)
			return nil, err
		}
		comics.Score = scores[ID]
		result = append(result, comics)
	}
	s.log.Debug("returning comics", "count", len(result))

	return result, nil
}

func (s *Service) BuildIndex(ctx context.Context) error {

	s.index.Clear()
	lastID, err := s.db.LastID(ctx)
	if err != nil {
		return err
	}
	var comicsCount int
	for ID := 1; ID <= lastID; ID++ {
		comics, err := s.db.Get(ctx, ID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			s.log.Error("failed to fetch comics", "id", ID, "error", err)
			return err
		}
		s.index.Put(ID, comics.Keywords)
		comicsCount++
	}

	s.log.Debug("rebuilt index", "comics count", comicsCount)
	return nil
}

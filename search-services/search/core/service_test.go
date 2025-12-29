package core

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var noopLogger = slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

type FakeWords struct {
	normalized []string
	err        error
}

func (fw *FakeWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	if fw.err != nil {
		return nil, fw.err
	}
	return fw.normalized, nil
}

type FakeDB struct {
	searchResults map[string][]int
	comics        map[int]Comics
	lastID        int
	searchErr     error
	getErr        error
	lastIDErr     error
}

func (fd *FakeDB) Search(ctx context.Context, keyword string) ([]int, error) {
	if fd.searchErr != nil {
		return nil, fd.searchErr
	}
	return fd.searchResults[keyword], nil
}

func (fd *FakeDB) Get(ctx context.Context, id int) (Comics, error) {
	if fd.getErr != nil {
		return Comics{}, fd.getErr
	}
	comics, exists := fd.comics[id]
	if !exists {
		return Comics{}, ErrNotFound
	}
	return comics, nil
}

func (fd *FakeDB) LastID(ctx context.Context) (int, error) {
	if fd.lastIDErr != nil {
		return 0, fd.lastIDErr
	}
	return fd.lastID, nil
}

func TestService_Search_HappyPath(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{
		searchResults: map[string][]int{
			"happy": {1, 2},
			"year":  {2},
		},
		comics: map[int]Comics{
			1: {ID: 1, URL: "http://xkcd.com/1", Keywords: []string{"happy"}},
			2: {ID: 2, URL: "http://xkcd.com/2", Keywords: []string{"happy", "year"}},
		},
	}
	words := &FakeWords{normalized: []string{"happy", "year"}}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	result, err := svc.Search(ctx, "happy year", 10)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, 2, result[0].ID)
	assert.Equal(t, 2, result[0].Score)
	assert.Equal(t, 1, result[1].ID)
	assert.Equal(t, 1, result[1].Score)
}

func TestService_Search_NormalizationError(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{}
	words := &FakeWords{err: errors.New("invalid phrase")}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	result, err := svc.Search(ctx, "invalid", 10)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Equal(t, "invalid phrase", err.Error())
}

func TestService_Search_DBSearchError(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{searchErr: errors.New("db unavailable")}
	words := &FakeWords{normalized: []string{"test"}}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	result, err := svc.Search(ctx, "test", 10)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Equal(t, "db unavailable", err.Error())
}

func TestService_Search_DBGetError(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{
		searchResults: map[string][]int{"test": {1}},
		getErr:        errors.New("fetch failed"),
	}
	words := &FakeWords{normalized: []string{"test"}}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	result, err := svc.Search(ctx, "test", 10)

	require.Error(t, err)
	require.Nil(t, result)
	assert.Equal(t, "fetch failed", err.Error())
}

func TestService_Search_LimitApplied(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{
		searchResults: map[string][]int{
			"tree": {1, 2, 3, 4, 5},
		},
		comics: map[int]Comics{
			1: {ID: 1, URL: "http://xkcd.com/1"},
			2: {ID: 2, URL: "http://xkcd.com/2"},
			3: {ID: 3, URL: "http://xkcd.com/3"},
			4: {ID: 4, URL: "http://xkcd.com/4"},
			5: {ID: 5, URL: "http://xkcd.com/5"},
		},
	}
	words := &FakeWords{normalized: []string{"tree"}}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	result, err := svc.Search(ctx, "tree", 2)

	require.NoError(t, err)
	require.Len(t, result, 2)
}

func TestService_SearchIndex_HappyPath(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{
		comics: map[int]Comics{
			1: {ID: 1, URL: "http://xkcd.com/1", Score: 1},
			2: {ID: 2, URL: "http://xkcd.com/2", Score: 2},
		},
	}
	words := &FakeWords{normalized: []string{"happy", "year"}}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	svc.index.Put(1, []string{"happy"})
	svc.index.Put(2, []string{"happy", "year"})

	result, err := svc.SearchIndex(ctx, "happy year", 10)

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, 2, result[0].ID)
	assert.Equal(t, 1, result[1].ID)
}

func TestService_BuildIndex_HappyPath(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{
		lastID: 2,
		comics: map[int]Comics{
			1: {ID: 1, URL: "http://xkcd.com/1", Keywords: []string{"happy"}},
			2: {ID: 2, URL: "http://xkcd.com/2", Keywords: []string{"new", "year"}},
		},
	}
	words := &FakeWords{}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	err = svc.BuildIndex(ctx)

	require.NoError(t, err)
	assert.Len(t, svc.index.Get("happy"), 1)
	assert.Len(t, svc.index.Get("new"), 1)
	assert.Len(t, svc.index.Get("year"), 1)
}

func TestService_BuildIndex_IgnoresNotFound(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{
		lastID: 3,
		comics: map[int]Comics{
			1: {ID: 1, Keywords: []string{"a"}},
			3: {ID: 3, Keywords: []string{"b"}},
		},
	}
	words := &FakeWords{}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	err = svc.BuildIndex(ctx)

	require.NoError(t, err)
	assert.Len(t, svc.index.Get("a"), 1)
	assert.Len(t, svc.index.Get("b"), 1)
}

func TestService_BuildIndex_LastIDError(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{lastIDErr: errors.New("db error")}
	words := &FakeWords{}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	err = svc.BuildIndex(ctx)

	require.Error(t, err)
	assert.Equal(t, "db error", err.Error())
}

func TestService_BuildIndex_GetError(t *testing.T) {
	ctx := context.Background()
	db := &FakeDB{
		lastID: 2,
		comics: map[int]Comics{
			1: {ID: 1, Keywords: []string{"a"}},
		},
		getErr: errors.New("fetch error"),
	}
	words := &FakeWords{}
	svc, err := NewService(noopLogger, db, words)
	require.NoError(t, err)

	err = svc.BuildIndex(ctx)

	require.Error(t, err)
	assert.Equal(t, "fetch error", err.Error())
}

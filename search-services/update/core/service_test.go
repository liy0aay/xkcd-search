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

type FakeDB struct {
	added       []Comics
	dropCalled  bool
	IDsResult   []int
	StatsResult DBStats
	ErrAdd      error
	ErrIDs      error
	ErrStats    error
	ErrDrop     error
}

func (f *FakeDB) Add(ctx context.Context, c Comics) error {
	if f.ErrAdd != nil {
		return f.ErrAdd
	}
	f.added = append(f.added, c)
	return nil
}

func (f *FakeDB) IDs(ctx context.Context) ([]int, error) {
	if f.ErrIDs != nil {
		return nil, f.ErrIDs
	}
	return f.IDsResult, nil
}

func (f *FakeDB) Drop(ctx context.Context) error {
	f.dropCalled = true
	return f.ErrDrop
}

func (f *FakeDB) Stats(ctx context.Context) (DBStats, error) {
	if f.ErrStats != nil {
		return DBStats{}, f.ErrStats
	}
	return f.StatsResult, nil
}

type FakeXKCD struct {
	lastID int
	comics map[int]XKCDInfo
	ErrGet error
	ErrID  error
}

func (f *FakeXKCD) LastID(ctx context.Context) (int, error) {
	if f.ErrID != nil {
		return 0, f.ErrID
	}
	return f.lastID, nil
}

func (f *FakeXKCD) Get(ctx context.Context, id int) (XKCDInfo, error) {
	if f.ErrGet != nil {
		return XKCDInfo{}, f.ErrGet
	}
	return f.comics[id], nil
}

type FakeWords struct {
	Err      error
	Returned map[int][]string
}

func (fw *FakeWords) Norm(ctx context.Context, phrase string) ([]string, error) {
	if fw.Err != nil {
		return nil, fw.Err
	}
	return []string{"word"}, nil
}

func TestService_Status(t *testing.T) {
	db := &FakeDB{}
	xkcd := &FakeXKCD{}
	words := &FakeWords{}
	svc, _ := NewService(noopLogger, db, xkcd, words, 1)

	assert.Equal(t, StatusIdle, svc.Status(context.Background()))
	svc.inProgress.Store(true)
	assert.Equal(t, StatusRunning, svc.Status(context.Background()))
}

func TestService_Drop(t *testing.T) {
	db := &FakeDB{}
	xkcd := &FakeXKCD{}
	words := &FakeWords{}
	svc, _ := NewService(noopLogger, db, xkcd, words, 1)

	err := svc.Drop(context.Background())
	require.NoError(t, err)
	assert.True(t, db.dropCalled)
}

func TestService_Stats(t *testing.T) {
	db := &FakeDB{StatsResult: DBStats{WordsTotal: 10}}
	xkcd := &FakeXKCD{lastID: 42}
	words := &FakeWords{}
	svc, _ := NewService(noopLogger, db, xkcd, words, 1)

	stats, err := svc.Stats(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 10, stats.DBStats.WordsTotal)
	assert.Equal(t, 42, stats.ComicsTotal)
}

func TestService_Update_HappyPath(t *testing.T) {
	db := &FakeDB{IDsResult: []int{1}}
	xkcd := &FakeXKCD{
		lastID: 3,
		comics: map[int]XKCDInfo{
			2: {ID: 2, URL: "url2", Description: "desc2"},
			3: {ID: 3, URL: "url3", Description: "desc3"},
		},
	}
	words := &FakeWords{}
	svc, _ := NewService(noopLogger, db, xkcd, words, 2)

	err := svc.Update(context.Background())
	require.NoError(t, err)

	addedIDs := []int{db.added[0].ID, db.added[1].ID}
	assert.ElementsMatch(t, []int{2, 3}, addedIDs)

	addedURLs := []string{db.added[0].URL, db.added[1].URL}
	assert.ElementsMatch(t, []string{"url2", "url3"}, addedURLs)
}

func TestService_Update_LockPreventsDoubleRun(t *testing.T) {
	db := &FakeDB{}
	xkcd := &FakeXKCD{}
	words := &FakeWords{}
	svc, _ := NewService(noopLogger, db, xkcd, words, 1)

	svc.lock.Lock()
	defer svc.lock.Unlock()
	err := svc.Update(context.Background())
	assert.Equal(t, ErrAlreadyExists, err)
}

func TestService_Update_Errors(t *testing.T) {
	db := &FakeDB{ErrIDs: errors.New("db error")}
	xkcd := &FakeXKCD{}
	words := &FakeWords{}
	svc, _ := NewService(noopLogger, db, xkcd, words, 1)

	err := svc.Update(context.Background())
	assert.Error(t, err)
}

func TestService_Update_XKCDError(t *testing.T) {
	db := &FakeDB{}
	xkcd := &FakeXKCD{ErrID: errors.New("xkcd error")}
	words := &FakeWords{}
	svc, _ := NewService(noopLogger, db, xkcd, words, 1)

	err := svc.Update(context.Background())
	assert.Error(t, err)
}

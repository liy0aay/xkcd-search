package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/liy0aay/xkcd-search/api/adapters/explainxkcd"
	"github.com/liy0aay/xkcd-search/api/core"
)

func encodeReply(w io.Writer, reply any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(reply); err != nil {
		return fmt.Errorf("could not encode reply: %v", err)
	}
	return nil
}

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reply := PingResponse{
			Replies: make(map[string]string),
		}
		for name, pinger := range pingers {
			if err := pinger.Ping(r.Context()); err != nil {
				reply.Replies[name] = "unavailable"
				log.Error("one of services is not available", "service", name, "error", err)
				continue
			}
			reply.Replies[name] = "ok"
		}
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

type Authenticator interface {
	Login(user, password string) (accessToken string, refreshToken string, err error)
	Verify(token string) error
	RefreshAccessToken(refreshToken string) (string, error)
}

type Login struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func NewLoginHandler(log *slog.Logger, auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var l Login
		if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
			log.Error("could not decode login form", "error", err)
			http.Error(w, "could not parse login data", http.StatusBadRequest)
			return
		}
		accessToken, refreshToken, err := auth.Login(l.Name, l.Password)
		if err != nil {
			log.Error("could not authenticate", "user", l.Name, "error", err)
			http.Error(w, "could not authenticate", http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    refreshToken,
			Path:     "/",
			MaxAge:   30 * 24 * 3600,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"access_token": accessToken,
		}); err != nil {
			log.Error("failed to write reply", "error", err)
		}
	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Update(r.Context()); err != nil {
			log.Error("error while update", "error", err)
			if errors.Is(err, core.ErrAlreadyExists) {
				http.Error(w, err.Error(), http.StatusAccepted)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

type UpdateStats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := updater.Stats(r.Context())
		if err != nil {
			log.Error("error while stats", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reply := UpdateStats{
			WordsTotal:    stats.WordsTotal,
			WordsUnique:   stats.WordsUnique,
			ComicsFetched: stats.ComicsFetched,
			ComicsTotal:   stats.ComicsTotal,
		}
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

type UpdateStatus struct {
	Status string `json:"status"`
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status, err := updater.Status(r.Context())
		if err != nil {
			log.Error("error while status", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reply := UpdateStatus{Status: string(status)}
		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := updater.Drop(r.Context()); err != nil {
			log.Error("error while drop", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

type Comics struct {
	ID    int    `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
	Alt   string `json:"alt"`
	Score int    `json:"score"`
}

type ComicsReply struct {
	Comics []Comics `json:"comics"`
	Total  int      `json:"total"`
}

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit int
		var err error
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				log.Error("wrong limit", "value", limitStr)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			if limit < 0 {
				log.Error("wrong limit", "value", limit)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
		}
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			log.Error("no phrase")
			http.Error(w, "no phrase", http.StatusBadRequest)
			return
		}

		comics, err := searcher.Search(r.Context(), phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				http.Error(w, "no comics found", http.StatusNotFound)
				return
			}
			log.Error("error while seaching", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reply := ComicsReply{
			Comics: make([]Comics, 0, len(comics)),
			Total:  len(comics),
		}
		for _, c := range comics {
			reply.Comics = append(reply.Comics, Comics{ID: c.ID, URL: c.URL, Title: c.Title, Alt: c.Alt, Score: c.Score})
		}

		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}

func NewSearchIndexHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var limit int
		var err error
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				log.Error("wrong limit", "value", limitStr)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			if limit < 0 {
				log.Error("wrong limit", "value", limit)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
		}
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			log.Error("no phrase")
			http.Error(w, "no phrase", http.StatusBadRequest)
			return
		}

		comics, err := searcher.SearchIndex(r.Context(), phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				http.Error(w, "no comics found", http.StatusNotFound)
				return
			}
			log.Error("error while seaching", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		reply := ComicsReply{
			Comics: make([]Comics, 0, len(comics)),
			Total:  len(comics),
		}
		for _, c := range comics {
			reply.Comics = append(reply.Comics, Comics{ID: c.ID, URL: c.URL, Title: c.Title, Alt: c.Alt, Score: c.Score})
		}

		if err := encodeReply(w, reply); err != nil {
			log.Error("cannot encode reply", "error", err)
		}
	}
}
func NewExplainHandler(log *slog.Logger, client *explainxkcd.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		explanation, err := client.Explain(r.Context(), id)
		if err != nil {
			log.Error("explain failed", "error", err)
			if errors.Is(err, core.ErrNotFound) {
				http.Error(w, "not found", http.StatusNotFound)
			} else {
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(explanation); err != nil {
			log.Error("failed to encode explanation response", "error", err)
		}
	}
}

func NewRefreshTokenHandler(log *slog.Logger, auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("refresh_token")
		if err != nil {
			log.Error("refresh token not found in cookie", "error", err)
			http.Error(w, "refresh token not found", http.StatusUnauthorized)
			return
		}

		newAccessToken, err := auth.RefreshAccessToken(cookie.Value)
		if err != nil {
			log.Error("could not refresh access token", "error", err)
			http.Error(w, "could not refresh token", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"access_token": newAccessToken,
		}); err != nil {
			log.Error("failed to write refresh response", "error", err)
		}
	}
}

func NewLogoutHandler(log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"message": "logged out",
		}); err != nil {
			log.Error("failed to write logout response", "error", err)
		}
	}
}

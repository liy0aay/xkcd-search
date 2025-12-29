package core

import "context"

type Normalizer interface {
	Norm(context.Context, string) ([]string, error)
}

type Pinger interface {
	Ping(context.Context) error
}

type Updater interface {
	Update(context.Context) error
	Stats(context.Context) (UpdateStats, error)
	Status(context.Context) (UpdateStatus, error)
	Drop(context.Context) error
}

type Searcher interface {
	Search(context.Context, string, int) ([]Comics, error)
	SearchIndex(context.Context, string, int) ([]Comics, error)
}

type Authenticator interface {
	Login(user, password string) (accessToken string, refreshToken string, err error)
	Verify(token string) error
	RefreshAccessToken(refreshToken string) (string, error)
}

type Explainer interface {
	Explain(ctx context.Context, id int) (ExplainXKCDInfo, error)
}

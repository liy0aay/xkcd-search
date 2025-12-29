package core

type ServiceStatus string

const (
	StatusRunning ServiceStatus = "running"
	StatusIdle    ServiceStatus = "idle"
)

type DBStats struct {
	WordsTotal    int
	WordsUnique   int
	ComicsFetched int
}

type ServiceStats struct {
	DBStats     DBStats
	ComicsTotal int
}

type Comics struct {
	ID    int
	URL   string
	Title string
	Alt   string
	Words []string
}

type XKCDInfo struct {
	ID          int
	URL         string
	Title       string
	Alt         string
	Description string
}

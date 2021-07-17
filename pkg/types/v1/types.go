package v1

type ID int64

type SyncStatus string

const (
	StatusUninitialized SyncStatus = "uninitialized"
	StatusOK            SyncStatus = "ok"
	StatusOffline       SyncStatus = "offline"
	StatusSynchronizing SyncStatus = "synchronizing"
	StatusError         SyncStatus = "error"
)

type ByID []ID

func (p ByID) Len() int {
	return len(p)
}

func (p ByID) Less(i, j int) bool {
	return p[i] < p[j]
}

func (p ByID) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

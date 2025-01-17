package types

import "errors"

var (
	ErrAlreadyUpdating = errors.New("an update is already in progress, cannot start another")
)

type Update struct {
	Baseline Baseline `json:"baseline"` // Baseline is the set of versions that are available to update to.
	Updating bool     `json:"updating"` // Updating is true if an update is currently in progress.
}

type Updater interface {
	CurrentVersion() (string, error)
	Install(version string) error
	IsInstalled() bool
	ID() string
}

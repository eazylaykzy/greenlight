package data

import (
	"database/sql"
	"errors"
)

// ErrRecordNotFound error. We'll return this from our Get() method when
// looking up a movie that doesn't exist in our database
var (
	ErrEditConflict   = errors.New("edit conflict")
	ErrRecordNotFound = errors.New("record not found")
)

// Models struct which wraps the MovieModel
type Models struct {
	Movies MovieModel
}

// NewModels For ease of use, we also add a New() method which returns a Models struct containing the initialized MovieModel
func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
	}
}

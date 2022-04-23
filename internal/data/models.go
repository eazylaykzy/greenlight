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

type Models struct {
	Movies MovieModel
	Users  UserModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Movies: MovieModel{DB: db},
		Users:  UserModel{DB: db}, // Initialize a new UserModel instance.
	}
}

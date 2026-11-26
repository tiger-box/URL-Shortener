package domain

import "errors"

type URL struct {
	Code        string
	OriginalURL string
}

var (
	ErrNotFound      = errors.New("url not found")
	ErrAlreadyExists = errors.New("url already exists")
	ErrZero          = errors.New("Url zero")
	ErrCode          = errors.New("could not find an available code")
)

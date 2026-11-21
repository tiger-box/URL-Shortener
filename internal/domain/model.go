package domain

import "errors"

type URL struct {
	Code        string
	OriginalURL string
}

var (
	ErrNotFound      = errors.New("url not found")
	ErrAlreadyExists = errors.New("url already exists")
)

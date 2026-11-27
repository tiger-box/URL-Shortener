package domain

import "errors"

// URL — короткая ссылка: код и исходный адрес, на который он ведёт.
// Новые поля записи (время создания, срок жизни) добавляются сюда,
// не меняя сигнатуры репозиториев.
type URL struct {
	Code        string
	OriginalURL string
}

var (
	ErrNotFound      = errors.New("url not found")
	ErrAlreadyExists = errors.New("code already exists")
	ErrInvalidURL    = errors.New("invalid url")
	ErrCodeConflict  = errors.New("could not find an available code")
)

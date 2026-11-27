// Package service содержит бизнес-правила сокращателя ссылок:
// проверку входных данных, генерацию кодов и повтор при конфликте.
// Хранение данных делегируется репозиторию через интерфейс.
package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"

	"github.com/tiger-box/URL-Shortener/internal/domain"
)

// repository — хранилище пар «код → URL», которое нужно сервису.
// Save возвращает domain.ErrAlreadyExists, если код занят;
// Get и Delete возвращают domain.ErrNotFound, если кода нет.
type repository interface {
	Save(ctx context.Context, code string, url string) error
	Get(ctx context.Context, code string) (string, error)
	Delete(ctx context.Context, code string) error
}

// chars — алфавит для кодов. Без похожих символов (0/O/o, 1/I/i/l),
// чтобы код можно было без ошибок переписать с экрана или продиктовать.
const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ" +
	"abcdefghjkmnpqrstuvwxyz" +
	"23456789"

// codeLen — длина короткого кода.
const codeLen = 5

// Shortener создаёт короткие ссылки и работает с ними.
// Безопасен для одновременного использования из нескольких горутин,
// если таким же является репозиторий.
type Shortener struct {
	repo repository
}

// NewShortener создаёт сервис поверх переданного репозитория.
func NewShortener(repo repository) *Shortener {
	return &Shortener{repo: repo}
}

// Create сохраняет url под новым случайным кодом и возвращает этот код.
// Пустой url — domain.ErrInvalidURL. Если сгенерированный код занят,
// генерирует новый и пробует снова.
func (s *Shortener) Create(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("create short url: %w", domain.ErrInvalidURL)
	}

	for {
		code := generateCode()
		err := s.repo.Save(ctx, code, url)

		if err == nil {
			return code, nil
		}

		if errors.Is(err, domain.ErrAlreadyExists) {
			continue
		}

		return "", fmt.Errorf("create short url: %w", err)
	}
}

// Get возвращает исходный URL по короткому коду.
// Пустой или несуществующий код — domain.ErrNotFound.
func (s *Shortener) Get(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", fmt.Errorf("get url: %w", domain.ErrNotFound)
	}

	url, err := s.repo.Get(ctx, code)
	if err != nil {
		return "", fmt.Errorf("get url: %w", err)
	}

	return url, nil
}

// Delete удаляет короткую ссылку. После удаления Get по этому коду
// возвращает domain.ErrNotFound. Пустой или несуществующий код — domain.ErrNotFound.
func (s *Shortener) Delete(ctx context.Context, code string) error {
	if code == "" {
		return fmt.Errorf("delete url: %w", domain.ErrNotFound)
	}

	err := s.repo.Delete(ctx, code)
	if err != nil {
		return fmt.Errorf("delete url: %w", err)
	}

	return nil
}

// generateCode возвращает случайный код длины codeLen из алфавита chars.
func generateCode() string {
	var builder strings.Builder
	for range codeLen {
		builder.WriteByte(chars[rand.IntN(len(chars))])
	}
	return builder.String()
}

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

const (
	// codeLen — длина короткого кода. 55⁵ ≈ 503 млн вариантов; когда
	// заполнение станет заметным (повторы учащаются), длину увеличивают на 1.
	codeLen = 5

	// maxAttempts — сколько раз Create пробует новый код, если сгенерированный
	// уже занят. Без лимита при заполненном хранилище запрос зависнет навсегда.
	// При заполнении 50% вероятность 10 неудач подряд — около 0,1%.
	maxAttempts = 10
)

// CodeGenerator возвращает кандидата в короткие коды.
// Уникальность не гарантируется: занятость кода проверяет репозиторий.
// Реализация должна быть безопасна для вызова из нескольких горутин.
type CodeGenerator func() string

// Option настраивает Shortener при создании.
type Option func(*Shortener)

// WithGenerator подменяет генератор кодов (например, на предсказуемый в тестах).
// nil игнорируется — остаётся генератор по умолчанию.
func WithGenerator(gen CodeGenerator) Option {
	return func(s *Shortener) {
		if gen != nil {
			s.generate = gen
		}
	}
}

// Shortener создаёт короткие ссылки и работает с ними.
// Безопасен для одновременного использования из нескольких горутин,
// если таким же является репозиторий.
type Shortener struct {
	repo     repository
	generate CodeGenerator
}

// NewShortener создаёт сервис поверх переданного репозитория.
// По умолчанию коды случайные; поведение меняется через опции.
func NewShortener(repo repository, opts ...Option) *Shortener {
	s := &Shortener{
		repo:     repo,
		generate: generateCode,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Create сохраняет url под новым случайным кодом и возвращает этот код.
// Пустой url — domain.ErrInvalidURL. Если сгенерированный код занят,
// генерирует новый и пробует снова, но не больше maxAttempts раз;
// после этого — domain.ErrCodeConflict. Перед каждой попыткой проверяет ctx:
// отменённый запрос возвращает ошибку контекста и не обращается к хранилищу.
func (s *Shortener) Create(ctx context.Context, url string) (string, error) {
	if url == "" {
		return "", fmt.Errorf("create short url: %w", domain.ErrInvalidURL)
	}

	for range maxAttempts {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("create short url: %w", err)
		}

		code := s.generate()
		err := s.repo.Save(ctx, code, url)

		if err == nil {
			return code, nil
		}

		if errors.Is(err, domain.ErrAlreadyExists) {
			continue
		}

		return "", fmt.Errorf("create short url: %w", err)
	}

	return "", fmt.Errorf("create short url: %w", domain.ErrCodeConflict)
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

// generateCode — генератор по умолчанию: случайный код длины codeLen
// из алфавита chars. Глобальные функции math/rand/v2 безопасны
// для конкурентного вызова и не требуют ручного seed.
func generateCode() string {
	var builder strings.Builder
	for range codeLen {
		builder.WriteByte(chars[rand.IntN(len(chars))])
	}
	return builder.String()
}

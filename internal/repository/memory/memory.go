// Package memory — хранилище коротких ссылок в оперативной памяти.
// Данные теряются при перезапуске; подходит для тестов, CLI и локального запуска.
package memory

import (
	"context"
	"sync"

	"github.com/tiger-box/URL-Shortener/internal/domain"
)

// Storage хранит пары «код → URL». Только хранит: не генерирует коды
// и не проверяет URL. Безопасен для одновременного использования из
// нескольких горутин.
type Storage struct {
	urls map[string]string
	mu   sync.RWMutex
}

// NewStorage создаёт пустое хранилище.
func NewStorage() *Storage {
	return &Storage{urls: make(map[string]string)}
}

// Save сохраняет url под кодом code. Если код уже занят — возвращает
// domain.ErrAlreadyExists и не перезаписывает существующую запись.
// Проверка и запись выполняются атомарно.
func (s *Storage) Save(_ context.Context, code, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.urls[code]; ok {
		return domain.ErrAlreadyExists
	}

	s.urls[code] = url
	return nil
}

// Get возвращает URL по коду или domain.ErrNotFound, если кода нет.
func (s *Storage) Get(_ context.Context, code string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	url, ok := s.urls[code]
	if !ok {
		return "", domain.ErrNotFound
	}

	return url, nil
}

// Delete удаляет запись по коду или возвращает domain.ErrNotFound, если кода нет.
func (s *Storage) Delete(_ context.Context, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.urls[code]; !ok {
		return domain.ErrNotFound
	}

	delete(s.urls, code)
	return nil
}

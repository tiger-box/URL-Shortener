// Package memory — хранилище коротких ссылок в оперативной памяти.
// Данные теряются при перезапуске; подходит для тестов, CLI и локального запуска.
package memory

import (
	"context"
	"sync"

	"github.com/tiger-box/URL-Shortener/internal/domain"
)

// Storage хранит короткие ссылки (domain.URL) с ключом по коду.
// Только хранит: не генерирует коды и не проверяет URL.
// Безопасен для одновременного использования из нескольких горутин.
type Storage struct {
	urls map[string]domain.URL
	mu   sync.RWMutex
}

// NewStorage создаёт пустое хранилище.
func NewStorage() *Storage {
	return &Storage{urls: make(map[string]domain.URL)}
}

// Save сохраняет запись u под кодом u.Code. Если код уже занят — возвращает
// domain.ErrAlreadyExists и не перезаписывает существующую запись.
// Проверка и запись выполняются атомарно.
func (s *Storage) Save(_ context.Context, u domain.URL) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.urls[u.Code]; ok {
		return domain.ErrAlreadyExists
	}

	s.urls[u.Code] = u
	return nil
}

// Get возвращает запись по коду. Если кода нет — пустую domain.URL
// и domain.ErrNotFound.
func (s *Storage) Get(_ context.Context, code string) (domain.URL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.urls[code]
	if !ok {
		return domain.URL{}, domain.ErrNotFound
	}

	return u, nil
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

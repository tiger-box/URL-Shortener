package memory_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/tiger-box/URL-Shortener/internal/domain"
	"github.com/tiger-box/URL-Shortener/internal/repository/memory"
)

// goDev — типовая запись для тестов.
var goDev = domain.URL{Code: "abc12", OriginalURL: "https://go.dev"}

// Get отдаёт запись целиком — с теми же Code и OriginalURL, что сохранили.
// Структура из строк сравнивается через ==.
func TestStorage_SaveAndGet(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStorage()

	if err := s.Save(ctx, goDev); err != nil {
		t.Fatalf("Save: unexpected error: %v", err)
	}

	got, err := s.Get(ctx, goDev.Code)
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got != goDev {
		t.Errorf("Get = %+v, want %+v", got, goDev)
	}
}

func TestStorage_SaveDuplicate(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStorage()

	if err := s.Save(ctx, goDev); err != nil {
		t.Fatalf("first Save: unexpected error: %v", err)
	}

	err := s.Save(ctx, domain.URL{Code: "abc12", OriginalURL: "https://example.com"})
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("second Save: got %v, want %v", err, domain.ErrAlreadyExists)
	}

	// Первая запись не должна быть перезаписана.
	got, _ := s.Get(ctx, goDev.Code)
	if got != goDev {
		t.Errorf("Get after duplicate Save = %+v, want %+v", got, goDev)
	}
}

// Ненайденная запись: ErrNotFound и пустая структура (zero value),
// а не «полузаполненная» — вызывающий не должен случайно получить мусор.
func TestStorage_NotFound(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStorage()

	got, err := s.Get(ctx, "nope1")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get missing: got %v, want %v", err, domain.ErrNotFound)
	}
	if got != (domain.URL{}) {
		t.Errorf("Get missing = %+v, want zero domain.URL", got)
	}
	if err := s.Delete(ctx, "nope1"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Delete missing: got %v, want %v", err, domain.ErrNotFound)
	}
}

func TestStorage_Delete(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStorage()

	if err := s.Save(ctx, goDev); err != nil {
		t.Fatalf("Save: unexpected error: %v", err)
	}
	if err := s.Delete(ctx, goDev.Code); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}
	if _, err := s.Get(ctx, goDev.Code); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get after Delete: got %v, want %v", err, domain.ErrNotFound)
	}
}

// Много горутин одновременно пишут и читают разные коды.
// Смысл теста — запуск с -race: без мьютекса детектор гонок упадёт.
func TestStorage_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStorage()

	const n = 100
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(2)
		code := fmt.Sprintf("c%04d", i)
		go func() {
			defer wg.Done()
			if err := s.Save(ctx, domain.URL{Code: code, OriginalURL: "https://go.dev"}); err != nil {
				t.Errorf("Save %s: %v", code, err)
			}
		}()
		go func() {
			defer wg.Done()
			_, _ = s.Get(ctx, code) // может ещё не быть — важен только одновременный доступ
		}()
	}
	wg.Wait()

	for i := range n {
		code := fmt.Sprintf("c%04d", i)
		if _, err := s.Get(ctx, code); err != nil {
			t.Errorf("Get %s after concurrent Save: %v", code, err)
		}
	}
}

// Много горутин одновременно сохраняют ОДИН И ТОТ ЖЕ код.
// Успеть должна ровно одна — проверка «занято?» и запись атомарны.
func TestStorage_ConcurrentSaveSameCode(t *testing.T) {
	ctx := context.Background()
	s := memory.NewStorage()

	const n = 100
	var wg sync.WaitGroup
	var ok, conflicts atomic.Int32
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.Save(ctx, domain.URL{Code: "same1", OriginalURL: fmt.Sprintf("https://site%d.com", i)})
			switch {
			case err == nil:
				ok.Add(1)
			case errors.Is(err, domain.ErrAlreadyExists):
				conflicts.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if ok.Load() != 1 {
		t.Errorf("successful saves = %d, want 1", ok.Load())
	}
	if conflicts.Load() != n-1 {
		t.Errorf("conflicts = %d, want %d", conflicts.Load(), n-1)
	}
}

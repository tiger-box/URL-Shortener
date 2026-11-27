package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tiger-box/URL-Shortener/internal/domain"
	"github.com/tiger-box/URL-Shortener/internal/service"
)

// fakeRepo — поддельный репозиторий для тестов сервиса.
// Хранит данные в map и умеет «притворяться» занятым или сломанным.
type fakeRepo struct {
	data      map[string]string
	conflicts int   // сколько первых Save вернут ErrAlreadyExists
	saveErr   error // если задано — Save всегда возвращает эту ошибку
	saveCalls int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{data: make(map[string]string)}
}

func (f *fakeRepo) Save(_ context.Context, code, url string) error {
	f.saveCalls++
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.conflicts > 0 {
		f.conflicts--
		return domain.ErrAlreadyExists
	}
	f.data[code] = url
	return nil
}

func (f *fakeRepo) Get(_ context.Context, code string) (string, error) {
	url, ok := f.data[code]
	if !ok {
		return "", domain.ErrNotFound
	}
	return url, nil
}

func (f *fakeRepo) Delete(_ context.Context, code string) error {
	if _, ok := f.data[code]; !ok {
		return domain.ErrNotFound
	}
	delete(f.data, code)
	return nil
}

// ТЗ §1–2: Create возвращает код, по которому Get отдаёт исходный URL.
func TestShortener_CreateThenGet(t *testing.T) {
	ctx := context.Background()
	s := service.NewShortener(newFakeRepo())

	code, err := s.Create(ctx, "https://example.com/some/long/path")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if code == "" {
		t.Fatal("Create returned empty code")
	}

	got, err := s.Get(ctx, code)
	if err != nil {
		t.Fatalf("Get(%q): unexpected error: %v", code, err)
	}
	if got != "https://example.com/some/long/path" {
		t.Errorf("Get(%q) = %q, want %q", code, got, "https://example.com/some/long/path")
	}
}

// ТЗ §1 и §8: коды уникальны, несколько ссылок живут одновременно
// и каждая ведёт на свой URL.
func TestShortener_CreateMany(t *testing.T) {
	ctx := context.Background()
	s := service.NewShortener(newFakeRepo())

	const n = 1000
	codes := make(map[string]string, n) // код → URL
	for i := range n {
		url := fmt.Sprintf("https://example.com/page/%d", i)
		code, err := s.Create(ctx, url)
		if err != nil {
			t.Fatalf("Create #%d: unexpected error: %v", i, err)
		}
		if prev, dup := codes[code]; dup {
			t.Fatalf("code %q issued twice: for %q and %q", code, prev, url)
		}
		codes[code] = url
	}

	for code, want := range codes {
		got, err := s.Get(ctx, code)
		if err != nil {
			t.Fatalf("Get(%q): unexpected error: %v", code, err)
		}
		if got != want {
			t.Errorf("Get(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestShortener_CreateEmptyURL(t *testing.T) {
	repo := newFakeRepo()
	s := service.NewShortener(repo)

	_, err := s.Create(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidURL) {
		t.Fatalf("got %v, want %v", err, domain.ErrInvalidURL)
	}
	if repo.saveCalls != 0 {
		t.Errorf("Save called %d times, want 0: invalid input must not reach storage", repo.saveCalls)
	}
}

// Код занят 3 раза подряд — сервис должен пробовать новые коды,
// а не вернуть ошибку и не вернуть несохранённый код.
func TestShortener_CreateRetriesOnConflict(t *testing.T) {
	repo := newFakeRepo()
	repo.conflicts = 3
	s := service.NewShortener(repo)

	code, err := s.Create(context.Background(), "https://go.dev")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if repo.saveCalls != 4 {
		t.Errorf("Save calls = %d, want 4 (3 conflicts + 1 success)", repo.saveCalls)
	}
	if got, err := s.Get(context.Background(), code); err != nil || got != "https://go.dev" {
		t.Errorf("Get(%q) = %q, %v; want %q, nil: returned code must be saved", code, got, err, "https://go.dev")
	}
}

// ТЗ §7 + план 1.3: если свободный код не находится, Create не крутится вечно,
// а сдаётся с domain.ErrCodeConflict.
func TestShortener_CreateGivesUpOnEndlessConflict(t *testing.T) {
	t.Skip("шаг 1.3: лимит попыток ещё не реализован — убрать Skip после 1.3")

	repo := newFakeRepo()
	repo.conflicts = 1 << 30 // склад всегда отвечает «занято»
	s := service.NewShortener(repo)

	type result struct {
		code string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		code, err := s.Create(context.Background(), "https://go.dev")
		done <- result{code, err}
	}()

	select {
	case r := <-done:
		if !errors.Is(r.err, domain.ErrCodeConflict) {
			t.Errorf("got %v, want %v", r.err, domain.ErrCodeConflict)
		}
		if r.code != "" {
			t.Errorf("code = %q, want empty on error", r.code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Create did not give up on endless conflict: infinite retry loop")
	}
}

// Любая ошибка, кроме «код занят», должна вернуться наверх, а не повторяться.
func TestShortener_CreateStorageError(t *testing.T) {
	errDB := errors.New("connection refused")
	repo := newFakeRepo()
	repo.saveErr = errDB
	s := service.NewShortener(repo)

	code, err := s.Create(context.Background(), "https://go.dev")
	if !errors.Is(err, errDB) {
		t.Fatalf("got %v, want wrapped %v", err, errDB)
	}
	if code != "" {
		t.Errorf("code = %q, want empty on error", code)
	}
	if repo.saveCalls != 1 {
		t.Errorf("Save calls = %d, want 1: non-conflict errors must not be retried", repo.saveCalls)
	}
}

func TestShortener_Get(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	repo.data["abc12"] = "https://go.dev"
	s := service.NewShortener(repo)

	got, err := s.Get(ctx, "abc12")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got != "https://go.dev" {
		t.Errorf("Get = %q, want %q", got, "https://go.dev")
	}

	for _, code := range []string{"", "nope1"} {
		if _, err := s.Get(ctx, code); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Get(%q): got %v, want %v", code, err, domain.ErrNotFound)
		}
	}
}

func TestShortener_Delete(t *testing.T) {
	ctx := context.Background()
	repo := newFakeRepo()
	repo.data["abc12"] = "https://go.dev"
	s := service.NewShortener(repo)

	if err := s.Delete(ctx, "abc12"); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}
	// ТЗ §3: после удаления получение по коду возвращает ошибку.
	if _, err := s.Get(ctx, "abc12"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Get after Delete: got %v, want %v", err, domain.ErrNotFound)
	}

	for _, code := range []string{"", "abc12"} {
		if err := s.Delete(ctx, code); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("Delete(%q): got %v, want %v", code, err, domain.ErrNotFound)
		}
	}
}

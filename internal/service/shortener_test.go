package service_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/tiger-box/URL-Shortener/internal/domain"
	"github.com/tiger-box/URL-Shortener/internal/repository/memory"
	"github.com/tiger-box/URL-Shortener/internal/service"
)

// fakeRepo — поддельный репозиторий для тестов сервиса.
// Хранит данные в map и умеет «притворяться» занятым или сломанным.
type fakeRepo struct {
	data      map[string]domain.URL
	conflicts int   // сколько первых Save вернут ErrAlreadyExists
	saveErr   error // если задано — Save всегда возвращает эту ошибку
	saveCalls int
	lastSaved domain.URL // что сервис передал в последний Save
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{data: make(map[string]domain.URL)}
}

func (f *fakeRepo) Save(_ context.Context, u domain.URL) error {
	f.saveCalls++
	f.lastSaved = u
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.conflicts > 0 {
		f.conflicts--
		return domain.ErrAlreadyExists
	}
	f.data[u.Code] = u
	return nil
}

func (f *fakeRepo) Get(_ context.Context, code string) (domain.URL, error) {
	u, ok := f.data[code]
	if !ok {
		return domain.URL{}, domain.ErrNotFound
	}
	return u, nil
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

// План 1.4: Create передаёт в репозиторий заполненную запись —
// Code совпадает с тем, что получил клиент, OriginalURL — с тем, что он прислал.
// Ловит ошибку «Save(ctx, domain.URL{})»: она компилируется, но теряет данные.
func TestShortener_CreateSavesFullRecord(t *testing.T) {
	repo := newFakeRepo()
	s := service.NewShortener(repo, service.WithGenerator(func() string { return "AAAAA" }))

	code, err := s.Create(context.Background(), "https://go.dev")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	want := domain.URL{Code: code, OriginalURL: "https://go.dev"}
	if repo.lastSaved != want {
		t.Errorf("saved record = %+v, want %+v", repo.lastSaved, want)
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

// План 1.3: генератор подменяется снаружи — сервис берёт коды из него,
// а не из встроенного случайного.
func TestShortener_WithGenerator(t *testing.T) {
	codes := []string{"AAAAA", "BBBBB"}
	next := 0
	gen := func() string {
		c := codes[next]
		next++
		return c
	}
	s := service.NewShortener(newFakeRepo(), service.WithGenerator(gen))

	for _, want := range codes {
		got, err := s.Create(context.Background(), "https://go.dev")
		if err != nil {
			t.Fatalf("Create: unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("Create = %q, want %q from injected generator", got, want)
		}
	}
}

// WithGenerator(nil) не должен оставить сервис без генератора (иначе panic).
func TestShortener_WithNilGenerator(t *testing.T) {
	s := service.NewShortener(newFakeRepo(), service.WithGenerator(nil))

	code, err := s.Create(context.Background(), "https://go.dev")
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if code == "" {
		t.Error("Create returned empty code: default generator must stay in place")
	}
}

// План 1.3: настоящий конфликт на настоящем хранилище. Генератор всегда
// выдаёт один и тот же код: первый Create его занимает, второй упирается
// в лимит попыток и получает ErrCodeConflict, а первая ссылка не затирается.
func TestShortener_CreateConflictWithMemoryRepo(t *testing.T) {
	ctx := context.Background()
	const fixed = "AAAAA"
	s := service.NewShortener(memory.NewStorage(),
		service.WithGenerator(func() string { return fixed }))

	code, err := s.Create(ctx, "https://first.example")
	if err != nil {
		t.Fatalf("first Create: unexpected error: %v", err)
	}
	if code != fixed {
		t.Fatalf("first Create = %q, want %q", code, fixed)
	}

	code, err = s.Create(ctx, "https://second.example")
	if !errors.Is(err, domain.ErrCodeConflict) {
		t.Fatalf("second Create: got %v, want %v", err, domain.ErrCodeConflict)
	}
	if code != "" {
		t.Errorf("second Create code = %q, want empty on error", code)
	}

	got, err := s.Get(ctx, fixed)
	if err != nil || got != "https://first.example" {
		t.Errorf("Get(%q) = %q, %v; want %q, nil: conflict must not overwrite existing link",
			fixed, got, err, "https://first.example")
	}
}

// Отменённый запрос не должен ходить в хранилище: клиент ушёл —
// генерировать и сохранять коды бессмысленно.
func TestShortener_CreateCanceledContext(t *testing.T) {
	repo := newFakeRepo()
	s := service.NewShortener(repo)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	code, err := s.Create(ctx, "https://go.dev")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want %v", err, context.Canceled)
	}
	if code != "" {
		t.Errorf("code = %q, want empty on error", code)
	}
	if repo.saveCalls != 0 {
		t.Errorf("Save calls = %d, want 0: canceled request must not reach storage", repo.saveCalls)
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
	repo.data["abc12"] = domain.URL{Code: "abc12", OriginalURL: "https://go.dev"}
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
	repo.data["abc12"] = domain.URL{Code: "abc12", OriginalURL: "https://go.dev"}
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

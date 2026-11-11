package internal

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
)

type URLShortener struct {
	urls map[string]string
	mu   sync.Mutex
}

func NewURLShortener() *URLShortener {
	return &URLShortener{urls: make(map[string]string)}
}

// https://example.com/some/long/path
func (s *URLShortener) Create(OriginalURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(OriginalURL) == 0 {
		return "", fmt.Errorf("shortener: len url empty")
	}

	//rand.Seed(time.Now().UnixNano())
	// Этот не сам делал, а просто нашел на сайте,
	// для генерации строки
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")

	var builder strings.Builder
	code := builder.String()

	for {
		for i := 0; i < 5; i++ {
			builder.WriteRune(chars[rand.IntN(len(chars))])
		}

		code = builder.String()
		if _, exitsts := s.urls[code]; !exitsts {
			break
		}
	}

	s.urls[code] = OriginalURL
	return code, nil
}

func (s *URLShortener) Get(urlsId string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	value, ok := s.urls[urlsId]
	if !ok {
		return "", errors.New("short URL not found")
	}

	return value, nil
}

func (s *URLShortener) Delete(shortUrl string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.urls[shortUrl]
	if !ok {
		// delete(url.urls, shortUrl)
		// return nil

		return errors.New("short URL not found")
	}

	delete(s.urls, shortUrl)
	return nil
}

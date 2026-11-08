package internal

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
)

type URLShortener struct {
	Urls map[string]string
}

func NewURLShortener() *URLShortener {
	return &URLShortener{Urls: make(map[string]string)}
}

// https://example.com/some/long/path
func (s *URLShortener) Create(OriginalURL string) (string, error) {
	if len(OriginalURL) == 0 {
		return "", fmt.Errorf("error:len url empty")
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

		if _, exitsts := s.Urls[code]; !exitsts {
			break
		}
	}

	s.Urls[code] = OriginalURL

	return code, nil
}

func (s *URLShortener) Get(urlsId string) (string, error) {
	value, ok := s.Urls[urlsId]
	if !ok {
		return "", errors.New("short URL not found")
	}

	return value, nil
}

func (s *URLShortener) Delete(shortUrl string) error {
	_, ok := s.Urls[shortUrl]
	if !ok {
		// delete(url.urls, shortUrl)
		// return nil

		return errors.New("short URL not found")
	}

	delete(s.Urls, shortUrl)
	return nil
}

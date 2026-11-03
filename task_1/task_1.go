package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

type URLShortener struct {
	urls map[string]string
}

func NewURLShortener() *URLShortener {
	return &URLShortener{urls: make(map[string]string)}
}

// https://example.com/some/long/path
func (url *URLShortener) Create(urls string) string {

	//rand.Seed(time.Now().UnixNano())
	// Этот не сам делал, а просто нашел на сайте,
	// для генерации строки
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")

	var bulder strings.Builder

	for i := 0; i < 5; i++ {
		bulder.WriteRune(chars[rand.IntN(len(chars))])
	}

	url.urls[bulder.String()] = urls

	return bulder.String()
}

func (url *URLShortener) Get(urlsId string) string {

	value, ok := url.urls[urlsId]
	if ok {
		return value
	}

	return ""
}

func (url *URLShortener) Delete(shortUrl string) error {

	_, ok := url.urls[shortUrl]
	if ok {
		delete(url.urls, shortUrl)
		return nil

	}
	return fmt.Errorf("short URL not found")
}
func main() {
	constructor := NewURLShortener()

	urL := "https://example.com/some/long/path"
	// url2 := "https://www.youtube.com/shorts/IXdGMPGO0kk"
	// urul3 := "https://mail.google.com/mail/u/0/?tab=rm&ogbl#inbox/FMfcgzQjggmmRTkQVnRPtwFVMwrjBFss"

	shortId := constructor.Create(urL)
	// shortId2 := constructor.Create(url2)
	// shortId3 := constructor.Create(urul3)

	// fmt.Println(shortId)
	// fmt.Println(shortId2)
	// fmt.Println(shortId3)

	// fmt.Println("оригинал", constructor.Get(shortId))
	// fmt.Println("оригинал", constructor.Get(shortId2))
	// fmt.Println("оригинал", constructor.Get(shortId3))

	// fmt.Println(constructor.Delete(shortId))

	// fmt.Println(constructor.urls)

	fmt.Println("Введите нумерацию")
	fmt.Println("1. create short URL")
	fmt.Println("2.Get Url")
	fmt.Println("3.Delete Url")
	fmt.Println("4.exit")

	fmt.Println("= = = = = = = = = =")

	var choice int

	for {
		switch choice {
		case 1:
			fmt.Println(">", choice)
			fmt.Println("URL: ", urL)
			fmt.Println("Short code", shortId)

		case 2:
			fmt.Println(">", choice)
			fmt.Println("Code: ", shortId)
			fmt.Println("URL: ", constructor.Get(shortId))

		case 3:
			fmt.Println("Delete", constructor.Delete(shortId))
			fmt.Println("Сылка удалена")
		case 4:
			fmt.Println("Exit")
			return
		}

		fmt.Scan(&choice)

	}

	fmt.Println(constructor)

}

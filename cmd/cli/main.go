package main

import (
	"context"
	"fmt"

	"github.com/tiger-box/URL-Shortener/internal/repository/memory"
	"github.com/tiger-box/URL-Shortener/internal/service"
)

func main() {

	fmt.Println("Enter only the numbers 1 ... 4.")
	fmt.Println("1. Create short URL")
	fmt.Println("2. Get URL")
	fmt.Println("3. Delete URL")
	fmt.Println("4. Exit")

	fmt.Println("==============")

	var number int
	ctx := context.Background()
	memRepo := memory.NewStorage()
	shortener := service.NewShortener(memRepo)

	fmt.Println("number choice")
	for {
		if _, err := fmt.Scan(&number); err != nil {
			fmt.Println("failed to add the number", err)
			return
		}

		switch number {
		case 1:
			var url string
			fmt.Print("Enter url: ")
			fmt.Scan(&url)
			short, err := shortener.Create(ctx, url)

			if err != nil {
				fmt.Println("error", err)
				continue
			}

			fmt.Println(short)

		case 2:
			fmt.Println("enter short url")

			var short string
			fmt.Scan(&short)
			url, err := shortener.Get(ctx, short)

			if err != nil {
				fmt.Println("error", err)
				continue
			}

			fmt.Println(url)

		case 3:
			fmt.Println("enter short url")

			var short string
			fmt.Scan(&short)
			err := shortener.Delete(ctx, short)

			if err != nil {
				fmt.Println("error", err)
				continue
			}

			fmt.Println("successfully deleted")

		case 4:
			fmt.Println("completed")
			return

		default:
			fmt.Println("wrong number")
		}
	}

	//fmt.Println(shortener.Urls)

}

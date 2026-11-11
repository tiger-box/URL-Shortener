package main

import (
	"fmt"

	"github.com/tiger-box/URL-Shortener/internal"
)

func main() {

	fmt.Println("Enter only the numbers 1 ... 4.")
	fmt.Println("1. Create short URL")
	fmt.Println("2. Get URL")
	fmt.Println("3. Delete URL")
	fmt.Println("4. Exit")

	fmt.Println("==============")

	var number int

	shortener := internal.NewURLShortener()

	fmt.Println("number choice")
	for {
		fmt.Scan(&number)
		switch number {
		case 1:
			var url string
			fmt.Print("Enter url: ")
			fmt.Scan(&url)
			create, err := shortener.Create(url)

			if err != nil {
				fmt.Println("error")
				return
			}

			fmt.Println(create)

		case 2:
			fmt.Println("enter short url")

			var id string
			fmt.Scan(&id)
			url, err := shortener.Get(id)

			if err != nil {
				fmt.Println("error")
				return
			}

			fmt.Println(url)

		case 3:
			fmt.Println("enter short url")

			var id string
			fmt.Scan(&id)
			err := shortener.Delete(id)

			if err != nil {
				fmt.Println("error")
				return
			}

			fmt.Println("successfully deleted")

		case 4:
			fmt.Println("completed")
			return
		}

	}

	//fmt.Println(shortener.Urls)

}

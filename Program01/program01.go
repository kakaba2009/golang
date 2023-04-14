package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/net/html"
)

func ParseTokens(resp *http.Response) {
	file, err := os.Create("result.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	tokenizer := html.NewTokenizer(resp.Body)

	for {
		tokenType := tokenizer.Next()

		switch tokenType {
		case html.ErrorToken:
			err := tokenizer.Err()
			log.Fatalf("error %v", err)
			break
		case html.TextToken:
			token := tokenizer.Token()
			x := token.Data
			if len(x) > 0 {
				fmt.Println(x)
				writer.Write([]string{"token", x})
			}
		case html.StartTagToken:
			token := tokenizer.Token()
			if "title" == token.Data {
				tokenType = tokenizer.Next()
				if tokenType == html.TextToken {
					x := tokenizer.Token().Data
					if len(x) > 0 {
						fmt.Println(x)
						writer.Write([]string{"title", x})
					}
				}
			} else {
				x := token.Data
				if len(x) > 0 {
					fmt.Println(x)
					writer.Write([]string{"token", x})
				}
			}
		}
	}
}

func ReadPage(link string) string {
	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	ParseTokens(res)
	res.Body.Close()
	return ""
}

func main() {
	ReadPage("https://www.secretchina.com")
}

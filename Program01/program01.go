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
	isTitle := false
	isLink := false

	for {
		tokenType := tokenizer.Next()

		switch tokenType {
		case html.ErrorToken:
			fmt.Println("Finished")
			return
		case html.StartTagToken:
			token := tokenizer.Token()
			if "title" == token.Data {
				isTitle = true
			} else if "a" == token.Data {
				isLink = true
			}
		case html.TextToken:
			if isTitle {
				ProcessText(tokenizer, writer)
				isTitle = false
			} else if isLink {
				ProcessText(tokenizer, writer)
				isLink = false
			}
		}
	}
}

func ProcessText(tokenizer *html.Tokenizer, writer *csv.Writer) {
	token := tokenizer.Token()
	data := token.Data
	if len(data) > 0 {
		fmt.Println(data)
		writer.Write([]string{data})
	}
}

func ReadPage(link string) {
	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}
	ParseTokens(res)
	res.Body.Close()
}

func main() {
	ReadPage("https://www.secretchina.com")
}

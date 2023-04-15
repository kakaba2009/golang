package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
)

func ParseTokens(resp *http.Response) {
	file, err := os.Create("result.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	ProcessText([]string{"id", "url", "title"}, writer)

	title := doc.Find("title").Text()
	ProcessText([]string{"title", "", title}, writer)

	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		ids, _ := s.Attr("id")
		if ids != "" {
			s.Find("a").Each(func(i int, s *goquery.Selection) {
				url, _ := s.Attr("href")
				txt, _ := s.Attr("title")
				ProcessText([]string{ids, url, txt}, writer)
			})
		}
	})
}

func ProcessText(data []string, writer *csv.Writer) {
	if len(data) > 0 {
		fmt.Println(data)
		writer.Write(data)
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

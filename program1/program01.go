package program1

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"

	"github.com/PuerkitoBio/goquery"
)

func ParseTokens(resp *http.Response) error {
	file, err := os.Create("result.csv")
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = ProcessText([]string{"id", "url", "title"}, writer)
	if err != nil {
		fmt.Println(err)
	}

	title := doc.Find("title").Text()
	err = ProcessText([]string{"title", "", title}, writer)
	if err != nil {
		fmt.Println(err)
	}

	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		ids, _ := s.Attr("id")
		if ids != "" {
			s.Find("a").Each(func(i int, s *goquery.Selection) {
				url, _ := s.Attr("href")
				txt, _ := s.Attr("title")
				err = ProcessText([]string{ids, url, txt}, writer)
				if err != nil {
					fmt.Println(err)
				}
			})
		}
	})

	return nil
}

func ProcessText(data []string, writer *csv.Writer) error {
	if len(data) > 0 {
		fmt.Println(data)
		return writer.Write(data)
	}
	return nil
}

func ReadPage(link string) error {
	res, err := http.Get(link)
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = ParseTokens(res)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return res.Body.Close()
}

func Main() error {
	return ReadPage("https://www.secretchina.com")
}

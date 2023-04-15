package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

var wg sync.WaitGroup
var hp string = "https://www.secretchina.com"

func FindLinks(resp *http.Response, job chan string) {
	defer wg.Done()
	tokenizer := html.NewTokenizer(resp.Body)
	isLink := false
	url := ""

	for {
		tokenType := tokenizer.Next()

		switch tokenType {
		case html.ErrorToken:
			fmt.Println("Finished")
			return
		case html.StartTagToken:
			token := tokenizer.Token()
			if "a" == token.Data {
				isLink = true
				for i := 0; i < len(token.Attr); i++ {
					attr := token.Attr[i]
					if attr.Key == "href" {
						url = attr.Val
						fmt.Println(url)
						break
					}
				}
			}
		case html.TextToken:
			if isLink {
				ProcessText(tokenizer, job, url)
				isLink = false
				url = ""
			}
		}
	}
}

func ProcessText(tokenizer *html.Tokenizer, job chan string, url string) {
	token := tokenizer.Token()
	data := token.Data
	if strings.TrimSpace(data) != "" && strings.TrimSpace(url) != "" {
		fmt.Println(data)
		job <- url + "|" + data
	}
}

func WriteFile(dir string, name string, content string) {
	f, err := os.Create(dir + "/" + name + ".html")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err2 := f.WriteString(content)
	if err2 != nil {
		log.Fatal(err2)
	}
	fmt.Println("done")
}

func ReadSubPage(job chan string, dir string) {
	defer wg.Done()
	for data := range job {
		links := strings.Split(data, "|")
		url := links[0]
		if strings.Contains(url, hp) || strings.Contains(url, "http:") || strings.Contains(url, "https:") {
			continue
		}
		name := links[1]
		res, err := http.Get(hp + url)
		if err != nil {
			log.Fatal(err)
		}
		content, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		WriteFile(dir, name, string(content))
	}
}

func ReadMainPage(link string, job chan string, dir string) {
	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	go FindLinks(res, job)

	threads := 1
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go ReadSubPage(job, dir)
	}

	wg.Wait()
	res.Body.Close()
}

func main() {
	job := make(chan string)
	dir := time.Now().Format("2006-01-02")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}
	ReadMainPage(hp, job, dir)
}

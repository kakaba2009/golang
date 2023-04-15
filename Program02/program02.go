package main

import (
	"crypto/md5"
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
	defer close(job)
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
	// Ignore other web page url links
	if strings.Contains(url, "http:") || strings.Contains(url, "https:") || strings.HasPrefix(url, "#") {
		return
	}
	token := tokenizer.Token()
	data := token.Data
	if strings.TrimSpace(data) != "" && strings.TrimSpace(url) != "" {
		jobData := url + "|" + data
		job <- jobData
		fmt.Println(jobData)
	}
}

func WriteFile(dir string, name string, content string) {
	md5s := md5.Sum([]byte(name))
	hash := fmt.Sprintf("%x", md5s)
	f, err := os.Create(dir + "/" + hash + ".html")
	if err != nil {
		log.Fatal(err)
		return
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
			return
		}
		content, err := ioutil.ReadAll(res.Body)
		WriteFile(dir, name, string(content))
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func ReadMainPage(link string, job chan string, dir string) {
	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
		return
	}

	wg.Add(1)
	go FindLinks(res, job)

	threads := 5
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

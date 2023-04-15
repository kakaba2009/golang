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
	fmt.Println("Start to find links ... ")
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
	fmt.Println("ProcessText ... ")
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

func IsDownloaded(dir string, name string) bool {
	full := hashName(name, dir)
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return false
	}
	return true
}

func hashName(name string, dir string) string {
	md5s := md5.Sum([]byte(name))
	hash := fmt.Sprintf("%x", md5s)
	full := dir + "/" + hash + ".txt"
	return full
}

func WriteFile(dir string, name string, content string) {
	full := hashName(name, dir)
	f, err := os.Create(full)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer f.Close()

	_, err2 := f.WriteString(content)
	if err2 != nil {
		log.Fatal(err2)
	}
	fmt.Println("WriteFile done")
}

func ReadSubPage(job chan string, dir string) {
	fmt.Println("ReadSubPage ... ")
	defer wg.Done()
	for data := range job {
		links := strings.Split(data, "|")
		url := links[0]
		if strings.Contains(url, hp) || strings.Contains(url, "http:") || strings.Contains(url, "https:") {
			continue
		}
		name := links[1]
		if IsDownloaded(dir, name) {
			fmt.Println(url + " already downaded, skip ...")
			continue
		}
		res, err := http.Get(hp + url)
		if err != nil {
			log.Fatal(err)
			continue
		}
		content, err := ioutil.ReadAll(res.Body)
		WriteFile(dir, name, string(content))
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func ReadMainPage(link string, dir string) {
	fmt.Println("ReadMainPage ... ")
	job := make(chan string)

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

func Download() {
	fmt.Println("Start to download ... ")
	dir := time.Now().Format("2006-01-02")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}
	ReadMainPage(hp, dir)
}

func main() {
	for {
		Download()
		fmt.Println("Sleep ...")
		time.Sleep(time.Minute)
	}
}

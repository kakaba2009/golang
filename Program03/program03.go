package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var wg sync.WaitGroup
var hp string = "https://www.secretchina.com"

func FindLinks(resp *http.Response, job chan string) {
	fmt.Println("Start to find links ... ")
	defer close(job)
	defer wg.Done()
	doc, err := goquery.NewDocumentFromReader(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		ids, _ := s.Attr("id")
		if ids != "" {
			s.Find("a").Each(func(i int, s *goquery.Selection) {
				url, _ := s.Attr("href")
				txt, _ := s.Attr("title")
				ProcessText(job, url, txt)
			})
		}
	})
}

func ProcessText(job chan string, url string, title string) {
	fmt.Println("ProcessText ... ")
	// Ignore other web page url links
	if strings.Contains(url, "http:") || strings.Contains(url, "https:") || strings.HasPrefix(url, "#") {
		return
	}
	if strings.TrimSpace(title) != "" && strings.TrimSpace(url) != "" {
		jobData := url + "|" + title
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
			fmt.Println(url + " already downloaded, skip ...")
			continue
		}
		res, err := http.Get(hp + url)
		if err != nil {
			log.Fatal(err)
			continue
		}
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Fatal(err)
			continue
		}
		content := doc.Find("p").Text()
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

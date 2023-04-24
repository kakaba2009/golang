package program2

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
	defer close(job)
	defer wg.Done()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
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

func WriteFile(dir string, name string, content string) error {
	md5s := md5.Sum([]byte(name))
	hash := fmt.Sprintf("%x", md5s)
	f, err := os.Create(dir + "/" + hash + ".txt")
	if err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		log.Println(err)
		return err
	}

	fmt.Println("done")
	return nil
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
			log.Println(err)
			continue
		}
		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Println(err)
			continue
		}
		content := doc.Find("p").Text()
		WriteFile(dir, name, string(content))
		err = res.Body.Close()
		if err != nil {
			log.Println(err)
		}
	}
}

func ReadMainPage(link string, job chan string, dir string) error {
	res, err := http.Get(link)
	if err != nil {
		fmt.Println(err)
		return err
	}

	wg.Add(1)
	go FindLinks(res, job)

	threads := 5
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go ReadSubPage(job, dir)
	}

	wg.Wait()
	return res.Body.Close()
}

func Main() error {
	job := make(chan string)
	dir := time.Now().Format("2006-01-02")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}
	return ReadMainPage(hp, job, dir)
}

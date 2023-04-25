package program5

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/kakaba2009/golang/global"
	"github.com/labstack/echo/v4"
)

type ConfigFile = global.ConfigFile

func FindLinks(resp *http.Response, job chan string, wg *sync.WaitGroup) error {
	log.Println("Start to find links ... ")
	file, err := os.Create("public/id_file.csv")
	if err != nil {
		log.Println(err)
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

	defer close(job)
	defer wg.Done()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println(err)
		return err
	}

	doc.Find("li").Each(func(i int, s *goquery.Selection) {
		ids, _ := s.Attr("id")
		if ids != "" {
			s.Find("a").Each(func(i int, s *goquery.Selection) {
				url, _ := s.Attr("href")
				txt, _ := s.Attr("title")
				ProcessText(job, url, txt, ids)
				writer.Write([]string{ids})
			})
		}
	})

	return nil
}

func ProcessText(job chan string, url string, title string, id string) {
	log.Println("ProcessText ... ")
	// Ignore other web page url links
	if strings.Contains(url, "http:") || strings.Contains(url, "https:") || strings.HasPrefix(url, "#") {
		return
	}
	if strings.TrimSpace(title) != "" && strings.TrimSpace(url) != "" {
		jobData := url + "|" + title + "|" + id
		job <- jobData
		log.Println(jobData)
	}
}

func IsDownloaded(dir string, name string) bool {
	full := FullName(dir, name)
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return false
	}
	return true
}

func FullName(dir string, name string) string {
	return dir + "/" + name + ".txt"
}

func WriteFile(dir string, name string, content string) error {
	full := FullName(dir, name)
	f, err := os.Create(full)
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
	log.Println("WriteFile done")
	return nil
}

func ReadSubPage(job chan string, dir string, config ConfigFile, wg *sync.WaitGroup) {
	log.Println("ReadSubPage ... ")
	defer wg.Done()
	for data := range job {
		links := strings.Split(data, "|")
		url := links[0]
		if strings.Contains(url, "http:") || strings.Contains(url, "https:") {
			continue
		}
		// Use ID as name for file save
		name := links[2]
		if IsDownloaded(dir, name) {
			log.Println(url + " already downloaded, skip ...")
			continue
		}
		res, err := http.Get(config.Url + url)
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
		err = WriteFile(dir, name, string(content))
		if err != nil {
			log.Println(err)
		}
		res.Body.Close()
	}
}

func ReadMainPage(link string, dir string, config ConfigFile) error {
	var wg sync.WaitGroup

	log.Println("ReadMainPage ... ")
	job := make(chan string)

	res, err := http.Get(link)
	if err != nil {
		log.Println(err)
		return err
	}

	wg.Add(1)
	go FindLinks(res, job, &wg)

	threads := config.Threads
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go ReadSubPage(job, dir, config, &wg)
	}

	wg.Wait()
	return res.Body.Close()
}

func Download(config ConfigFile) error {
	log.Println("Start to download ... ")
	dir := "public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}
	os.Create(dir + "/id_file.csv")

	err := ReadMainPage(config.Url, dir, config)
	if err != nil {
		log.Println(err)
		return err
	}
	// Generate html index
	return GenerateHtml()
}

func StartEcho() error {
	e := echo.New()
	e.Static("/", "public")
	return e.Start(":8000")
}

func Main() error {
	pwd, _ := os.Getwd()
	log.Println(pwd)

	file := "program5/config.json"

	if len(os.Args) >= 2 {
		// Use config file from command line
		file = os.Args[1]
		log.Println("Use config file " + file)
	}

	conFile, err := os.ReadFile(file)
	if err != nil {
		log.Print(err)
		return err
	}
	var config ConfigFile
	err = json.Unmarshal(conFile, &config)
	log.Println(config)

	go PeriodicDownload(config)
	// Start Web Server
	return StartEcho()
}

func PeriodicDownload(config ConfigFile) {
	ticker := time.NewTicker(time.Minute * time.Duration(config.Interval))
	for {
		select {
		case t := <-ticker.C:
			log.Println("Ticking at", t)
			Download(config)
		}
	}
}

func GenerateHtml() error {
	csv, err := os.ReadFile("public/id_file.csv")
	if err != nil {
		log.Println(err)
		return err
	}
	all_ids := string(csv)
	ids := strings.Split(all_ids, "\n")

	html := `
	<!DOCTYPE html>
	<html>
	<head>
	<title>ECHO Web Server</title>
	</head>
	<body>
	<h1>Article</h1>`

	for i := 0; i < len(ids); i++ {
		if ids[i] != "" {
			html += "<li><a href=" + ids[i] + ".txt>" + ids[i] + "</a></li>"
		}
	}

	html += `</body>
	</html>`

	f, err := os.Create("public/index.html")
	if err != nil {
		log.Println(err)
		return err
	}
	defer f.Close()
	_, err = f.WriteString(html)
	if err != nil {
		log.Println(err)
	}
	return err
}

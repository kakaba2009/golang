package program8

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kakaba2009/golang/program7"
	"github.com/kakaba2009/golang/program8/handler"
	"github.com/labstack/echo/v4"
)

var wg sync.WaitGroup

type ConfigFile struct {
	Url      string `json:"url"`
	Threads  int    `json:"threads"`
	Interval int    `json:"interval"`
}

type Record struct {
	Id    string
	Title string
	Url   string
}

type TemplateRegistry struct {
	templates *template.Template
}

func FindLinks(resp *http.Response, job chan string, db *sql.DB) {
	fmt.Println("Start to find links ... ")

	file, err := os.Create("public/id_file.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()

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
				ProcessText(job, url, txt, ids)
				writer.Write([]string{ids})
				WriteToDatabase(db, ids, txt, url)
			})
		}
	})
}

func WriteToDatabase(db *sql.DB, id string, title string, url string) {
	// Delete the same id row if exists
	del := "DELETE FROM article WHERE id = '" + id + "'"
	_, err1 := db.Exec(del)
	if err1 != nil {
		log.Fatal(err1)
	}

	sql := "INSERT INTO article(id, title, url) VALUES ('" + id + "', '" + title + "', '" + url + "')"
	_, err2 := db.Exec(sql)
	if err2 != nil {
		log.Fatal(err2)
	}
}

func ProcessText(job chan string, url string, title string, id string) {
	fmt.Println("ProcessText ... ")
	// Ignore other web page url links
	if strings.Contains(url, "http:") || strings.Contains(url, "https:") || strings.HasPrefix(url, "#") {
		return
	}
	if strings.TrimSpace(title) != "" && strings.TrimSpace(url) != "" {
		jobData := url + "|" + title + "|" + id
		job <- jobData
		fmt.Println(jobData)
	}
}

func IsDownloaded(dir string, name string) bool {
	full := fullName(dir, name)
	if _, err := os.Stat(full); os.IsNotExist(err) {
		return false
	}
	return true
}

func fullName(dir string, name string) string {
	return dir + "/" + name + ".txt"
}

func hashName(name string, dir string) string {
	md5s := md5.Sum([]byte(name))
	hash := fmt.Sprintf("%x", md5s)
	full := dir + "/" + hash + ".txt"
	return full
}

func WriteFile(dir string, name string, content string) {
	full := fullName(dir, name)
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

func ReadSubPage(job chan string, dir string, config ConfigFile) {
	fmt.Println("ReadSubPage ... ")
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
			fmt.Println(url + " already downloaded, skip ...")
			continue
		}
		res, err := http.Get(config.Url + url)
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

func ReadMainPage(link string, dir string, config ConfigFile, db *sql.DB) {
	fmt.Println("ReadMainPage ... ")
	job := make(chan string)

	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
		return
	}

	wg.Add(1)
	go FindLinks(res, job, db)

	threads := config.Threads
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go ReadSubPage(job, dir, config)
	}

	wg.Wait()
	res.Body.Close()
}

func Download(config ConfigFile, db *sql.DB) {
	fmt.Println("Start to download ... ")
	dir := "public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}
	os.Create(dir + "/id_file.csv")

	ReadMainPage(config.Url, dir, config, db)
	// Generate html index
	// GenerateHtml(db)
}

func Main(args []string) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "config.json"

	if len(args) >= 2 {
		// Use config file from command line
		file = args[1]
		fmt.Println("Use config file " + file)
	}

	conFile, err := os.ReadFile(file)
	if err != nil {
		fmt.Print(err)
		return
	}
	var config ConfigFile
	err = json.Unmarshal(conFile, &config)
	fmt.Println(config)

	db, err0 := sql.Open("mysql", "golang:3306@tcp(127.0.0.1:3306)/golang")
	defer db.Close()
	if err0 != nil {
		log.Fatal(err0)
	}

	// Start Web Server
	e := StartEcho()

	// Below function blocks
	timerDownload(config, quit, db)

	//graceful shutdown ECHO
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
	fmt.Println("Exiting ECHO ...")
}

func timerDownload(config ConfigFile, quit chan os.Signal, db *sql.DB) {
	defer fmt.Println("Exiting timer download")
	ticker := time.NewTicker(time.Minute * time.Duration(config.Interval))
	for {
		select {
		case t := <-ticker.C:
			fmt.Println("Ticking at", t)
			Download(config, db)
		case <-quit:
			fmt.Println("Received CTRL+C, exiting ...")
			return
		}
	}
}

func GenerateHtml(db *sql.DB) {
	ids := program7.GetIdsFromDatabase(db)

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
		log.Fatal(err)
		return
	}
	defer f.Close()
	_, err2 := f.WriteString(html)
	if err2 != nil {
		log.Fatal(err2)
	}
}

func StartEcho() *echo.Echo {
	e := echo.New()
	e.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseGlob("public/*.html")),
	}

	// Named route "golang"
	e.GET("/", handler.HomeHandler)
	e.Static("/public", "public")
	// Start server
	go func() {
		if err := e.Start(":8000"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down ECHO server")
		}
	}()

	return e
}

func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

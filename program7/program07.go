package program7

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program5"
	"github.com/labstack/echo/v4"
)

type ConfigFile = global.ConfigFile

type Record = global.Record

func FindLinks(resp *http.Response, job chan string, db *sql.DB, wg *sync.WaitGroup) {
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
				program5.ProcessText(job, url, txt, ids)
				writer.Write([]string{ids})
				WriteToDatabase(db, ids, txt, url)
			})
		}
	})
}

func WriteToDatabase(db *sql.DB, id string, title string, url string) {
	// Delete the same id row if exists
	del := "DELETE FROM article WHERE id = ?"
	_, err1 := db.Exec(del, id)
	if err1 != nil {
		log.Fatal(err1)
	}

	sql := "INSERT INTO article(id, title, url) VALUES (?, ?, ?)"
	_, err2 := db.Exec(sql, id, title, url)
	if err2 != nil {
		log.Fatal(err2)
	}
}

func ReadMainPage(link string, dir string, config ConfigFile, db *sql.DB) {
	var wg sync.WaitGroup

	fmt.Println("ReadMainPage ... ")
	job := make(chan string)

	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
		return
	}

	wg.Add(1)
	go FindLinks(res, job, db, &wg)

	threads := config.Threads
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go program5.ReadSubPage(job, dir, config, &wg)
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
	GenerateHtml(db)
}

func StartEcho() *echo.Echo {
	e := echo.New()
	e.Static("/", "public")
	// Start server
	go func() {
		if err := e.Start(":8000"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down ECHO server")
		}
	}()
	return e
}

func Main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "program7/config.json"

	if len(os.Args) >= 2 {
		// Use config file from command line
		file = os.Args[1]
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

func GetIdsFromDatabase(db *sql.DB) []string {
	sql := "SELECT id FROM article"
	res, err1 := db.Query(sql)
	if err1 != nil {
		log.Fatal(err1)
	}

	var ids []string
	for res.Next() {
		var row Record
		err2 := res.Scan(&row.Id)
		if err2 != nil {
			log.Fatal(err2)
		}
		ids = append(ids, row.Id)
	}
	return ids
}

func GenerateHtml(db *sql.DB) {
	ids := GetIdsFromDatabase(db)

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

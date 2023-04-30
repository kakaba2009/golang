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

func FindLinks(resp *http.Response, job chan string, db *sql.DB, wg *sync.WaitGroup) error {
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
				program5.ProcessText(job, url, txt, ids)
				writer.Write([]string{ids})
				err := WriteToDatabase(db, ids, txt, url)
				if err != nil {
					log.Println(err)
				}
			})
		}
	})

	return nil
}

func WriteToDatabase(db *sql.DB, id string, title string, url string) error {
	// Delete the same id row if exists
	del := "DELETE FROM article WHERE id = ?"
	_, err := db.Exec(del, id)
	if err != nil {
		return err
	}

	sql := "INSERT INTO article(id, title, url) VALUES (?, ?, ?)"
	_, err = db.Exec(sql, id, title, url)
	if err != nil {
		return err
	}

	return nil
}

func ReadMainPage(link string, dir string, config ConfigFile, db *sql.DB) error {
	var wg sync.WaitGroup

	fmt.Println("ReadMainPage ... ")
	job := make(chan string)

	res, err := http.Get(link)
	if err != nil {
		log.Println(err)
		return err
	}

	wg.Add(1)
	go FindLinks(res, job, db, &wg)

	threads := config.Threads
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go program5.ReadSubPage(job, dir, config, &wg)
	}

	wg.Wait()
	return res.Body.Close()
}

func Download(config ConfigFile, db *sql.DB) error {
	log.Println("Start to download ... ")
	dir := "public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}
	os.Create(dir + "/id_file.csv")

	err := ReadMainPage(config.Url, dir, config, db)
	if err != nil {
		return err
	}
	// Generate html index
	return GenerateHtml(db)
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

func Main() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "program7/config.json"

	if len(os.Args) >= 2 {
		// Use config file from command line
		file = os.Args[1]
		log.Println("Use config file " + file)
	}

	conFile, err := os.ReadFile(file)
	if err != nil {
		log.Println(err)
		return err
	}
	var config ConfigFile
	err = json.Unmarshal(conFile, &config)
	log.Println(config)

	db, err := sql.Open("mysql", "golang:3306@tcp(127.0.0.1:3306)/golang")
	defer db.Close()
	if err != nil {
		log.Println(err)
		return err
	}

	// Start Web Server
	e := StartEcho()

	// Below function blocks
	PeriodicDownload(config, quit, db)

	//graceful shutdown ECHO
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Print(err)
		return err
	}

	log.Println("Exiting ECHO Server ...")

	return nil
}

func PeriodicDownload(config ConfigFile, quit chan os.Signal, db *sql.DB) {
	defer fmt.Println("Exiting periodic download")
	ticker := time.NewTicker(time.Minute * time.Duration(config.Interval))
	for {
		select {
		case t := <-ticker.C:
			log.Println("Ticking at", t)
			Download(config, db)
		case <-quit:
			log.Println("Received CTRL+C, exiting ...")
			return
		}
	}
}

func GetIdsFromDatabase(db *sql.DB) ([]string, error) {
	sql := "SELECT id FROM article"
	res, err := db.Query(sql)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	var ids []string
	for res.Next() {
		var row Record
		err = res.Scan(&row.Id)
		if err != nil {
			log.Fatal(err)
		}
		ids = append(ids, row.Id)
	}
	return ids, nil
}

func GenerateHtml(db *sql.DB) error {
	ids, ok := GetIdsFromDatabase(db)
	if ok != nil {
		return ok
	}

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
		return err
	}

	return nil
}

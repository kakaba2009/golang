package program8

import (
	"context"
	"database/sql"
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
	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program5"
	"github.com/kakaba2009/golang/program7"
	"github.com/kakaba2009/golang/program8/handler"
	"github.com/labstack/echo/v4"
)

type ConfigFile = global.ConfigFile

type Record = global.Record

type TemplateRegistry struct {
	templates *template.Template
}

func FindLinks(resp *http.Response, job chan string, db *sql.DB, wg *sync.WaitGroup) error {
	log.Println("Start to find links ... ")
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
				program7.WriteToDatabase(db, ids, txt, url)
			})
		}
	})

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
		if program5.IsDownloaded(dir, name) {
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
		err = program5.WriteFile(dir, name, string(content))
		if err != nil {
			log.Println(err)
		}
		res.Body.Close()
	}
}

func ReadMainPage(link string, dir string, config ConfigFile, db *sql.DB) error {
	var wg sync.WaitGroup

	log.Println("ReadMainPage ... ")
	job := make(chan string)

	res, err := http.Get(link)
	if err != nil {
		return err
	}

	wg.Add(1)
	go FindLinks(res, job, db, &wg)

	threads := config.Threads
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go ReadSubPage(job, dir, config, &wg)
	}

	wg.Wait()
	return res.Body.Close()
}

func Download(config ConfigFile, db *sql.DB) error {
	fmt.Println("Start to download ... ")
	dir := "program8/public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}

	return ReadMainPage(config.Url, dir, config, db)
}

func Main() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "program8/config.json"

	if len(os.Args) >= 2 {
		// Use config file from command line
		file = os.Args[1]
		fmt.Println("Use config file " + file)
	}

	conFile, err := os.ReadFile(file)
	if err != nil {
		log.Print(err)
		return err
	}
	var config ConfigFile
	err = json.Unmarshal(conFile, &config)
	fmt.Println(config)

	db, err0 := sql.Open("mysql", "golang:3306@tcp(127.0.0.1:3306)/golang")
	defer db.Close()
	if err0 != nil {
		log.Println(err0)
		return err0
	}

	// Start Web Server
	e := StartEcho()

	// Below function blocks
	PeriodicDownload(config, quit, db)

	//graceful shutdown ECHO
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Println(err)
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
			err := Download(config, db)
			if err != nil {
				log.Println(err)
			}
		case <-quit:
			log.Println("Received CTRL+C, exiting ...")
			return
		}
	}
}

func StartEcho() *echo.Echo {
	e := echo.New()
	e.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseGlob("program8/public/*.html")),
	}

	e.GET("/", handler.HomeHandler)
	e.Static("/public", "program8/public")
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

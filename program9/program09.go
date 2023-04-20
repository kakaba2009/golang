package program9

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
	"github.com/kakaba2009/golang/program7"
	"github.com/kakaba2009/golang/program8"
	"github.com/kakaba2009/golang/program9/handler"
	"github.com/labstack/echo/v4"
)

var wg sync.WaitGroup

type ConfigFile struct {
	Url      string `json:"url"`
	Threads  int    `json:"threads"`
	Interval int    `json:"interval"`
}

type TemplateRegistry struct {
	templates *template.Template
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
		if program7.IsDownloaded(dir, name) {
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
		program8.WriteFile(dir, name, string(content))
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
	go program8.FindLinks(res, job, db)

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
	dir := "program9/public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}

	ReadMainPage(config.Url, dir, config, db)
}

func Main() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "program9/config.json"

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

func StartEcho() *echo.Echo {
	e := echo.New()
	e.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseGlob("program9/public/*.html")),
	}

	e.GET("/", handler.CookieHandler)
	e.Static("/public", "program9/public")
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

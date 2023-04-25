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
	"syscall"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program8"
	"github.com/kakaba2009/golang/program9/cookiehandler"
	"github.com/labstack/echo/v4"
)

type ConfigFile = global.ConfigFile

type TemplateRegistry struct {
	templates *template.Template
}

func Download(config ConfigFile, db *sql.DB) error {
	fmt.Println("Start to download ... ")
	dir := "program9/public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}

	return program8.ReadMainPage(config.Url, dir, config, db)
}

func Main() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "program9/config.json"

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

	fmt.Println("Exiting ECHO ...")
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

func StartEcho() *echo.Echo {
	e := echo.New()
	e.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseGlob("program9/public/*.html")),
	}

	e.GET("/", cookiehandler.CookieHandler)
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

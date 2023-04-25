package program6

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program5"
	"github.com/labstack/echo/v4"
)

type ConfigFile = global.ConfigFile

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

	file := "program6/config.json"

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

	// Start Web Server
	e := StartEcho()

	// Below function blocks
	PeriodicDownload(config, quit)

	//graceful shutdown ECHO
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
		return err
	}
	log.Println("Exiting ECHO Server ...")

	return nil
}

func PeriodicDownload(config ConfigFile, quit chan os.Signal) error {
	defer log.Println("Exiting periodic download")
	ticker := time.NewTicker(time.Minute * time.Duration(config.Interval))
	for {
		select {
		case t := <-ticker.C:
			log.Println("Ticking at", t)
			err := program5.Download(config)
			if err != nil {
				log.Println(err)
			}
		case <-quit:
			fmt.Println("Received CTRL+C, exiting ...")
			return nil
		}
	}
}

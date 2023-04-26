package program11

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
	"strconv"
	"syscall"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program10"
	"github.com/kakaba2009/golang/program7"
	"github.com/kakaba2009/golang/program8"
	"github.com/kakaba2009/golang/program9/cookiehandler"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

type ConfigFile = global.ConfigFile

type Article = global.Article

type ArticleData = global.ArticleData

type TemplateRegistry struct {
	templates *template.Template
}

var db *sql.DB
var ctx = context.Background()
var rdb *redis.Client

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func Download(config ConfigFile, db *sql.DB) error {
	fmt.Println("Start to download ... ")
	dir := "program11/public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}

	return program8.ReadMainPage(config.Url, dir, config, db)
}

func Main() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "program11/config.json"

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

	db, _ = sql.Open("mysql", "golang:3306@tcp(127.0.0.1:3306)/golang")
	defer db.Close()

	// Start Web Server
	e := StartWebServer()

	// Below function blocks
	PeriodicAction(config, quit, db)

	//graceful shutdown ECHO
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Println(err)
		return err
	}
	fmt.Println("Exiting ECHO Server ...")
	return nil
}

func PeriodicAction(config ConfigFile, quit chan os.Signal, db *sql.DB) {
	defer log.Println("Exiting Periodic Action")
	ticker := time.NewTicker(time.Minute * time.Duration(config.Interval))
	for {
		select {
		case t := <-ticker.C:
			log.Println("Ticking at", t)
			Download(config, db)
			err := PeriodicUpdateRedis(db)
			if err != nil {
				log.Println(err)
			}
		case <-quit:
			log.Println("Received CTRL+C, exiting ...")
			return
		}
	}
}

func StartWebServer() *echo.Echo {
	e := echo.New()
	e.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseGlob("program11/public/*.html")),
	}

	e.GET("/", RedisHandler)
	e.GET("/articles", GetArticles)
	e.DELETE("/articles/:id", DeleteArticle)
	e.POST("/articles/:id", UpdateArticle)
	e.Static("/public", "program11/public")
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

func GetArticlesFromRedis(db *sql.DB) ([]Article, error) {
	var data []Article

	val, err := rdb.Get(ctx, "articles").Result()

	if err == redis.Nil {
		// Does not exist in Redis yet
		data, _ = program10.GetArticlesFromDatabase(db)
		// Update redis in-memory data
		json, err := json.Marshal(data)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		err = rdb.Set(ctx, "articles", json, 0).Err()
		if err != nil {
			log.Println(err)
			return nil, err
		}
		return data, nil
	}

	json.Unmarshal([]byte(val), &data)

	return data, nil
}

// GetArticles responds with the list of all articles as JSON.
func GetArticles(c echo.Context) error {
	articles, err := GetArticlesFromRedis(db)
	if err != nil {
		return c.JSON(http.StatusNotAcceptable, err.Error())
	}
	return c.JSON(http.StatusOK, articles)
}

func DeleteArticleFromDatabase(db *sql.DB, id string) (string, error) {
	sql := "SELECT title FROM golang.article WHERE id = ?"
	res := db.QueryRow(sql, id)

	var row Article
	res.Scan(&row.Title)

	del := "DELETE FROM golang.article WHERE id = ?"
	_, err2 := db.Exec(del, id)
	if err2 != nil {
		log.Fatal(err2)
		return row.Title, err2
	}

	// Delete the data from Redis as well
	rdb.Del(ctx, "articles/"+id)

	return row.Title, nil
}

func DeleteArticle(c echo.Context) error {
	id := c.Param("id")
	title, err := DeleteArticleFromDatabase(db, id)
	if err != nil {
		return c.JSON(http.StatusNotAcceptable, err.Error())
	}
	return c.JSON(http.StatusOK, title)
}

func UpdateArticleFromDatabase(db *sql.DB, id string, a Article) (Article, error) {
	sql := "UPDATE golang.article SET title = ? WHERE id = ?"
	res, err := db.Exec(sql, a.Title, id)
	row, _ := res.RowsAffected()
	fmt.Println("Rows affected " + strconv.FormatInt(row, 10))

	// Update the data from Redis as well
	json, _ := json.Marshal(a)
	rdb.Set(ctx, "articles/"+id, json, 0)

	return a, err
}

func UpdateArticle(c echo.Context) error {
	id := c.Param("id")
	var objRequest Article
	if err := c.Bind(&objRequest); err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	article, err := UpdateArticleFromDatabase(db, id, objRequest)
	if err != nil {
		return c.JSON(http.StatusNotAcceptable, err.Error())
	}
	return c.JSON(http.StatusOK, article)
}

func GetIdsFromRedis(db *sql.DB) ([]string, error) {
	var data []string
	var err error
	var val string

	// Lookup ids in Redis first
	val, err = rdb.Get(ctx, "ids").Result()

	if err != nil {
		// Does not exist in Redis yet
		data, err = program7.GetIdsFromDatabase(db)
		if err != nil {
			return nil, err
		}
		// Update redis in-memory data
		var jobj []byte
		jobj, err = json.Marshal(data)
		if err != nil {
			return nil, err
		}
		err = rdb.Set(ctx, "ids", jobj, 0).Err()
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	err = json.Unmarshal([]byte(val), &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func RedisHandler(c echo.Context) error {
	ip := cookiehandler.CheckClientCookie(c)
	// If client cookie does not have IP, then set cookie
	if ip == "" {
		cookiehandler.SetClientCookie(c)
	}
	ids, err := GetIdsFromRedis(db)
	if err != nil {
		return c.JSON(http.StatusNotAcceptable, err.Error())
	}
	// Please note the the second parameter "index.html" is the template name and should
	// be equal to the value stated in the {{ define }} statement in "public/index.html"
	return c.Render(http.StatusOK, "index.html", ArticleData{
		Title:       "Article",
		ArticleList: ids,
	})
}

func PeriodicUpdateRedis(db *sql.DB) error {
	data, err := program10.GetArticlesFromDatabase(db)
	if err != nil {
		return err
	}
	// Update redis in-memory data
	json1, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = rdb.Set(ctx, "articles", json1, 0).Err()
	if err != nil {
		return err
	}

	ids, err := program7.GetIdsFromDatabase(db)
	if err != nil {
		return err
	}
	// Update redis in-memory data
	json2, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	rdb.Set(ctx, "ids", json2, 0)
	return nil
}

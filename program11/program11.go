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
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program10"
	"github.com/kakaba2009/golang/program5"
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

func ReadSubPage(job chan string, dir string, config ConfigFile, wg *sync.WaitGroup) {
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
		program8.WriteFile(dir, name, string(content))
		res.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func ReadMainPage(link string, dir string, config ConfigFile, db *sql.DB, wg *sync.WaitGroup) {
	fmt.Println("ReadMainPage ... ")
	job := make(chan string)

	res, err := http.Get(link)
	if err != nil {
		log.Fatal(err)
		return
	}

	wg.Add(1)
	go program8.FindLinks(res, job, db, wg)

	threads := config.Threads
	wg.Add(threads)
	for i := 1; i <= threads; i++ {
		go ReadSubPage(job, dir, config, wg)
	}

	wg.Wait()
	res.Body.Close()
}

func Download(config ConfigFile, db *sql.DB, wg *sync.WaitGroup) {
	fmt.Println("Start to download ... ")
	dir := "program11/public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
	}

	ReadMainPage(config.Url, dir, config, db, wg)
}

func Main() {
	var wg sync.WaitGroup

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "program11/config.json"

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

	db, _ = sql.Open("mysql", "golang:3306@tcp(127.0.0.1:3306)/golang")
	defer db.Close()

	// Start Web Server
	e := StartWebServer()

	// Below function blocks
	PeriodicAction(config, quit, db, &wg)

	//graceful shutdown ECHO
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
	fmt.Println("Exiting ECHO ...")
}

func PeriodicAction(config ConfigFile, quit chan os.Signal, db *sql.DB, wg *sync.WaitGroup) {
	defer fmt.Println("Exiting timer download")
	ticker := time.NewTicker(time.Minute * time.Duration(config.Interval))
	for {
		select {
		case t := <-ticker.C:
			fmt.Println("Ticking at", t)
			Download(config, db, wg)
			PeriodicUpdateRedis(db)
		case <-quit:
			fmt.Println("Received CTRL+C, exiting ...")
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

func GetArticlesFromRedis(db *sql.DB) []Article {
	var data []Article

	val, err := rdb.Get(ctx, "articles").Result()

	if err == redis.Nil {
		// Does not exist in Redis yet
		data = program10.GetArticlesFromDatabase(db)
		// Update redis in-memory data
		json, err := json.Marshal(data)
		if err != nil {
			fmt.Println(err)
		}
		err = rdb.Set(ctx, "articles", json, 0).Err()
		if err != nil {
			fmt.Println(err)
		}
		return data
	}

	json.Unmarshal([]byte(val), &data)

	return data
}

// GetArticles responds with the list of all articles as JSON.
func GetArticles(c echo.Context) error {
	articles := GetArticlesFromRedis(db)
	c.JSON(http.StatusOK, articles)
	return nil
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

func GetIdsFromRedis(db *sql.DB) []string {
	var data []string

	// Lookup ids in Redis first
	val, err := rdb.Get(ctx, "ids").Result()

	if err != nil {
		// Does not exist in Redis yet
		data, _ = program7.GetIdsFromDatabase(db)
		// Update redis in-memory data
		json, err := json.Marshal(data)
		if err != nil {
			fmt.Println(err)
		}
		err = rdb.Set(ctx, "ids", json, 0).Err()
		if err != nil {
			fmt.Println(err)
		}
		return data
	}

	json.Unmarshal([]byte(val), &data)

	return data
}

func RedisHandler(c echo.Context) error {
	ip := cookiehandler.CheckClientCookie(c)
	// If client cookie does not have IP, then set cookie
	if ip == "" {
		cookiehandler.SetClientCookie(c)
	}
	ids := GetIdsFromRedis(db)
	// Please note the the second parameter "index.html" is the template name and should
	// be equal to the value stated in the {{ define }} statement in "public/index.html"
	return c.Render(http.StatusOK, "index.html", ArticleData{
		Title:       "Article",
		ArticleList: ids,
	})
}

func PeriodicUpdateRedis(db *sql.DB) {
	var data []Article

	data = program10.GetArticlesFromDatabase(db)
	// Update redis in-memory data
	json1, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}
	err = rdb.Set(ctx, "articles", json1, 0).Err()
	if err != nil {
		fmt.Println(err)
	}

	ids, _ := program7.GetIdsFromDatabase(db)
	// Update redis in-memory data
	json2, _ := json.Marshal(ids)
	rdb.Set(ctx, "ids", json2, 0)
}

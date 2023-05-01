package program12

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v4"
	"github.com/kakaba2009/golang/cache"
	"github.com/kakaba2009/golang/database"
	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program11"
	"github.com/kakaba2009/golang/program8"
	"github.com/kakaba2009/golang/program9/cookiehandler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type ConfigFile = global.ConfigFile

type Article = global.Article

type ArticleData = global.ArticleData

type TemplateRegistry struct {
	templates *template.Template
}

type jwtCustomClaims struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

var ctx = context.Background()
var myKey = []byte("secret_key")

var savedPwd = map[string]string{
	"john": "hello!",
	"bill": "morning",
}

func Download(config ConfigFile, db *sql.DB) error {
	log.Println("Start to download ... ")
	dir := "program12/public"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, 0755)
		if err != nil {
			return err
		}
	}

	return program8.ReadMainPage(config.Url, dir, config, db)
}

func Main() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	file := "program12/config.json"

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
	if err != nil {
		log.Println(err)
		return err
	}
	log.Println(config)

	db := database.DB()
	defer db.Close()

	rdb := cache.Redis()
	defer rdb.Close()

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
	log.Println("Exiting ECHO Server ...")
	return nil
}

func PeriodicAction(config ConfigFile, quit chan os.Signal, db *sql.DB) {
	defer log.Println("Exiting Periodic Action")
	ticker := time.NewTicker(time.Minute * time.Duration(config.Interval))
	for {
		select {
		case t := <-ticker.C:
			log.Println("Ticking at", t)
			err := Download(config, db)
			if err != nil {
				log.Println(err)
			}
			err = program11.PeriodicUpdateRedis(db)
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
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Renderer = &TemplateRegistry{
		templates: template.Must(template.ParseGlob("program12/public/*.html")),
	}

	// Login route
	e.GET("/", RootHandler)
	e.POST("/login", LoginHandler)
	e.GET("/home", RedisHandler)
	e.GET("/articles", GetArticles)
	e.DELETE("/articles/:id", DeleteArticle)
	e.POST("/articles/:id", UpdateArticle)
	e.GET("/public/*", ArticleHandler)

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

// GetArticles responds with the list of all articles as JSON.
func GetArticles(c echo.Context) error {
	articles, err := program11.GetArticlesFromRedis(database.DB())
	if err != nil {
		return c.JSON(http.StatusNotAcceptable, err.Error())
	}
	return c.JSON(http.StatusOK, articles)
}

func DeleteArticle(c echo.Context) error {
	id := c.Param("id")
	title, err := program11.DeleteArticleFromDatabase(database.DB(), id)
	if err != nil {
		return c.JSON(http.StatusNotAcceptable, err.Error())
	}
	return c.JSON(http.StatusOK, title)
}

func UpdateArticle(c echo.Context) error {
	id := c.Param("id")
	var objRequest Article
	if err := c.Bind(&objRequest); err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, err.Error())
	}
	article, err := program11.UpdateArticleFromDatabase(database.DB(), id, objRequest)
	if err != nil {
		return c.JSON(http.StatusNotAcceptable, err.Error())
	}
	return c.JSON(http.StatusOK, article)
}

func RedisHandler(c echo.Context) error {
	token, err := GetTokenCookie(c)
	if err != nil {
		return c.Redirect(http.StatusMovedPermanently, "/")
	}
	// If client token cookie not valid, redirect to login page
	if !IsValidateToken(token) {
		return c.Redirect(http.StatusMovedPermanently, "/")
	}
	ids, err := program11.GetIdsFromRedis(database.DB())
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

func LoginHandler(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	pwd, found := savedPwd[username]
	if found == false {
		return echo.ErrUnauthorized
	}
	if password != pwd {
		return echo.ErrUnauthorized
	}

	// Set custom claims
	claims := &jwtCustomClaims{
		username,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	tkn, err := token.SignedString(myKey)
	if err != nil {
		return echo.ErrUnauthorized
	}
	log.Print("token ", tkn)

	// Set JWT token in client cookie
	SetTokenCookie(c, tkn)

	return c.Redirect(http.StatusMovedPermanently, "/home")
}

func SetTokenCookie(c echo.Context, tkn string) {
	cookie := new(http.Cookie)
	cookie.Name = "token"
	cookie.Value = tkn
	cookie.Expires = time.Now().Add(24 * time.Hour)
	c.SetCookie(cookie)
}

func GetTokenCookie(c echo.Context) (string, error) {
	cookie, err := c.Cookie("token")
	if err != nil {
		return "", err
	}
	return cookie.Value, nil
}

func RootHandler(c echo.Context) error {
	ip := cookiehandler.CheckClientCookie(c)
	// If client cookie does not have IP, then set cookie
	if ip == "" {
		cookiehandler.SetClientCookie(c)
	}
	// Please note the the second parameter "index.html" is the template name and should
	// be equal to the value stated in the {{ define }} statement in "public/index.html"
	return c.Render(http.StatusOK, "login.html", ArticleData{
		Title:       "Article",
		ArticleList: nil,
	})
}

func ArticleHandler(c echo.Context) error {
	token, err := GetTokenCookie(c)
	if err != nil {
		return c.Redirect(http.StatusMovedPermanently, "/")
	}
	// If client token cookie not valid, redirect to login page
	if !IsValidateToken(token) {
		return c.Redirect(http.StatusMovedPermanently, "/")
	}
	// Please note the the second parameter "index.html" is the template name and should
	// be equal to the value stated in the {{ define }} statement in "public/index.html"
	return c.File("program12/" + c.Request().RequestURI)
}

func IsValidateToken(token string) bool {
	claims := &jwtCustomClaims{}

	tknObj, err := jwt.ParseWithClaims(token, claims,
		func(t *jwt.Token) (interface{}, error) {
			return myKey, nil
		})

	if err != nil {
		return false
	}

	return tknObj.Valid
}

package handler

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kakaba2009/golang/program7"
	"github.com/labstack/echo/v4"
)

type ArticleData struct {
	Title       string
	ArticleList []string
}

var db *sql.DB
var err error

func init() {
	db, err = sql.Open("mysql", "golang:3306@tcp(127.0.0.1:3306)/golang")
	if err != nil {
		log.Fatal(err)
	}
}

func SetClientCookie(c echo.Context) {
	cookie := new(http.Cookie)
	cookie.Name = "IP"
	cookie.Value = c.RealIP()
	cookie.Expires = time.Now().Add(24 * time.Hour)
	c.SetCookie(cookie)
	fmt.Println("Set Client Cookie: ", cookie)
}

func CheckClientCookie(c echo.Context) string {
	cookie, err := c.Cookie("IP")
	if err != nil {
		return ""
	}
	fmt.Println("Cookie name: " + cookie.Name)
	fmt.Println("Cookir value " + cookie.Value)
	return cookie.Value
}

func CookieHandler(c echo.Context) error {
	ip := CheckClientCookie(c)
	// If client cookie does not have IP, then set cookie
	if ip == "" {
		SetClientCookie(c)
	}
	ids := program7.GetIdsFromDatabase(db)
	// Please note the the second parameter "index.html" is the template name and should
	// be equal to the value stated in the {{ define }} statement in "public/index.html"
	return c.Render(http.StatusOK, "index.html", ArticleData{
		Title:       "Article",
		ArticleList: ids,
	})
}

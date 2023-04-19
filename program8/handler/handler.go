package handler

import (
	"database/sql"
	"log"
	"net/http"

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

func HomeHandler(c echo.Context) error {
	ids := program7.GetIdsFromDatabase(db)
	// Please note the the second parameter "index.html" is the template name and should
	// be equal to the value stated in the {{ define }} statement in "public/index.html"
	return c.Render(http.StatusOK, "index.html", ArticleData{
		Title:       "Article",
		ArticleList: ids,
	})
}

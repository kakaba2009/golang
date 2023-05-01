package handler

import (
	"net/http"

	"github.com/kakaba2009/golang/database"
	"github.com/kakaba2009/golang/global"
	"github.com/kakaba2009/golang/program7"
	"github.com/labstack/echo/v4"
)

type ArticleData = global.ArticleData

func HomeHandler(c echo.Context) error {
	ids, _ := program7.GetIdsFromDatabase(database.DB())
	// Please note the the second parameter "index.html" is the template name and should
	// be equal to the value stated in the {{ define }} statement in "public/index.html"
	return c.Render(http.StatusOK, "index.html", ArticleData{
		Title:       "Article",
		ArticleList: ids,
	})
}

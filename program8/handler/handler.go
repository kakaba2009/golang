package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func HomeHandler(c echo.Context) error {
	// Please note the the second parameter "index.html" is the template name and should
	// be equal to the value stated in the {{ define }} statement in "public/index.html"
	return c.Render(http.StatusOK, "index.html", map[string]interface{}{
		"name": "ECHO Web Server",
		"msg":  "Article",
	})
}

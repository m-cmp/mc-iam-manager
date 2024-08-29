package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	e := echo.New()

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
	e.Renderer = renderer

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Static("/static", "assets")

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", map[string]interface{}{
			"name": "World",
		})
	})

	e.Any("/proxy/:domain/:method/*", func(c echo.Context) error {
		domain := c.Param("domain")
		method := strings.ToUpper(c.Param("method"))
		endpoint := c.Param("*")

		url := "http://" + domain + endpoint

		reqBody, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to read request body")
		}

		client := &http.Client{}
		req, err := http.NewRequest(method, url, strings.NewReader(string(reqBody)))
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to create request")
		}

		for header, values := range c.Request().Header {
			for _, value := range values {
				req.Header.Add(header, value)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to make request")
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Failed to read response")
		}

		return c.Blob(resp.StatusCode, resp.Header.Get("Content-Type"), body)
	})

	e.Logger.Fatal(e.Start(":8888"))
}

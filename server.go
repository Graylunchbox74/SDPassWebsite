package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"os"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var db *sql.DB
var tpl *template.Template

func init() {
	var err error
	_, ok := os.LookupEnv("NODB")
	if !ok {
		if err != nil {
			panic(err)
		}

		err = db.Ping()
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("No Database being used")
	}
}

func main() {
	r := gin.Default()
	tpl = template.Must(template.New("").ParseGlob("www/*.html"))

	//Route for the static files in www/
	r.Use(static.Serve("/www", static.LocalFile("www/", true)))

	r.GET("/", func(c *gin.Context) {

		tpl.ExecuteTemplate(c.Writer, "index.html", nil)
	})

	r.GET("/about", func(c *gin.Context) {

		tpl.ExecuteTemplate(c.Writer, "about.html", nil)
	})

	panic(r.Run(":6600"))
}

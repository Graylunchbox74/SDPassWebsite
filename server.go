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

const (
	dbhost = "DBHOST"
	dbport = "DBPORT"
	dbuser = "DBUSER"
	dbpass = "DBPASS"
	dbname = "DBNAME"
)

var db *sql.DB
var tpl *template.Template
var errorChannel chan locationalError

type locationalError struct {
	Error                 error
	Location, Sublocation string
}

//checking error functions
//check if there was an error
func checkErr(err error) {
	if err != nil {
		println(err)
	}
}

func checkLogError(location, sublocation string, err error) {
	if err != nil {
		logError(location, sublocation, err)
	}
}

func logError(location, sublocation string, err error) {
	errorChannel <- locationalError{err, location, sublocation}
}

func errorDrain() {
	var lErr locationalError
	for {
		select {
		case lErr = <-errorChannel:
			fmt.Println(lErr.Location, lErr.Sublocation, lErr.Error)
			//Handle Error Logging Here
		}
	}
}

func init() {
	var err error
	_, ok := os.LookupEnv("NODB")
	if !ok {
		db, err = sql.Open("sqlite3", "./userDatabase.db?_busy_timeout=5000")
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
	go errorDrain()
	r := gin.Default()

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

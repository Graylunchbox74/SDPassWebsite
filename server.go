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

func init() {
	var err error
	tpl = template.Must(template.New("").ParseGlob("www/*.html"))
	config := dbConfig()

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config[dbhost], config[dbport], config[dbuser], config[dbpass], config[dbname],
	)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println(fmt.Sprintf("Successfully connected to the %s database!", config[dbname]))
}

func main() {
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

func dbConfig() map[string]string {
	conf := make(map[string]string)
	conflist := []string{dbhost, dbport, dbuser, dbpass, dbname}
	for _, config := range conflist {
		con, ok := os.LookupEnv(config)
		if !ok {
			panic(fmt.Sprintf("%s environment variable required but not set", config))
		}
		conf[config] = con
	}
	return conf
}

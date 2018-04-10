package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type locationalError struct {
	Error                 error
	Location, Sublocation string
}

type internship struct {
	id                                                         int
	companyLogo, company, position, description, location, pay string
}

var db *sql.DB
var errorChannel chan locationalError
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

	r.GET("/login", func(c *gin.Context) {
		tpl.ExecuteTemplate(c.Writer, "login.html", nil)
	})

	panic(r.Run(":6600"))
}

func checkLogError(location, sublocation string, err error) {
	if err != nil {
		logError(location, sublocation, err)
	}
}

func errorDrain() {
	var lErr locationalError
	for {
		select {
		case lErr = <-errorChannel:
			fmt.Println(lErr.Location, lErr.Sublocation, lErr)
			//Handle Error Logging Here
		}
	}
}

func logError(location, sublocation string, err error) {
	errorChannel <- locationalError{err, location, sublocation}
}

func isActiveSession(r *http.Request) bool {
	funcLocation := "isActiveSession"
	val, err := r.Cookie("uuid")

	if err == nil {
		var uid uint
		err = db.QueryRow(`
			SELECT 
			uid 
			FROM USER_SESSIONS 
			WHERE uuid=$1`, val.Value,
		).Scan(&uid)

		if err != sql.ErrNoRows {
			if err == nil {
				return true
			}
			go logError(funcLocation, "1", err)
		}
	}
	return false
}

func addInternship(newInternship internship) error {
	var location = "AddInternship"
	var err error

	_, err = db.Exec("INSERT INTO <tablename> (companyLogo, company, position, description, location, pay) values($1,$2,$3,$4,$5,$6)", newInternship.companyLogo, newInternship.company, newInternship.position, newInternship.description, newInternship.location, newInternship.pay)
	checkLogError(location, "Exec for new internship", err)

	return err
}

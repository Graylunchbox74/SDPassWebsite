package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
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

	r.GET("/admin/add_internship", func(c *gin.Context) {
		checkAuth(c, func(c *gin.Context) {
			tpl.ExecuteTemplate(c.Writer, "internships.html", nil)
		})
	})

	r.POST("/admin/add_internship", func(c *gin.Context) {
		checkAuth(c, func(c *gin.Context) {
			//HANDLE ADDING INTERNSHIP HERE
		})
	})

	r.GET("/login", func(c *gin.Context) {
		if isActiveSession(c.Request) {
			tpl.ExecuteTemplate(c.Writer, "error.html", "You are already logged in!")
		} else {
			tpl.ExecuteTemplate(c.Writer, "login.html", nil)
		}
	})

	r.POST("/login", func(c *gin.Context) {
		var suser, spassword string
		user := strings.ToLower(c.PostForm("username"))
		password := c.PostForm("password")

		err := db.QueryRow(`
			SELECT username, password 
			FROM USERS 
			WHERE username=$1`, user,
		).Scan(
			&suser, &spassword,
		)

		if err == sql.ErrNoRows {
			go checkLogError(c.Request.URL.String(), "1", tpl.ExecuteTemplate(c.Writer, "login.html", "BAD LOGIN!"))
		} else {
			if err != nil {
				go logError(c.Request.URL.String(), "2", err)
				go checkLogError(c.Request.URL.String(), "3", tpl.ExecuteTemplate(c.Writer, "login.html", "ERROR LOGGING IN!"))
			} else {
				if checkPasswordHash(password, spassword) {
					uid := getUUID()
					http.SetCookie(c.Writer, &http.Cookie{Name: "uuid", Value: uid})

					_, err = db.Exec("INSERT INTO USER_SESSIONS (pid, uuid) VALUES ($1, $2)", user, uid)

					if err != nil {
						go logError(c.Request.URL.String(), "4", err)
						go checkLogError(c.Request.URL.String(), "5", tpl.ExecuteTemplate(c.Writer, "login.html", "ERROR LOGGING IN!"))
					} else {
						c.Redirect(303, "/")
					}
				} else {
					go checkLogError(c.Request.URL.String(), "7", tpl.ExecuteTemplate(c.Writer, "login.html", "BAD LOGIN!"))
				}
			}
		}
	})

	panic(r.Run(":6600"))
}

func checkAuth(c *gin.Context, f func(*gin.Context)) {
	if isActiveSession(c.Request) {
		f(c)
	} else {
		c.Redirect(303, "/")
	}
}

func checkLogError(location, sublocation string, err error) {
	if err != nil {
		logError(location, sublocation, err)
	}
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
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

func getUUID() string {
	var err error
	var uid uuid.UUID
	for uid, err = uuid.NewV4(); err != nil; {
		uid, err = uuid.NewV4()
	}
	return uid.String()
}

func logError(location, sublocation string, err error) {
	errorChannel <- locationalError{err, location, sublocation}
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
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

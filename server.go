package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

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

type program struct {
	id int
	companyLogo, company, position, description, location, majors, jobTitle,
	expirationOfPosting, contactInfo, typeOfProgram, startDate, endDate string
	pay float32
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

	r.GET("/admin/add_program", func(c *gin.Context) {
		checkAuth(c, func(c *gin.Context) {
			tpl.ExecuteTemplate(c.Writer, "internships.html", nil)
		})
	})

	r.POST("/admin/add_program", func(c *gin.Context) {
		checkAuth(c, func(c *gin.Context) {
			//I would have liked this to be the area where we
			//send back a confirmation page; will do it in the fucking future however
			//get data from the request
			//add a limit to the size of file
			companyLogo, err := c.FormFile("companyPhoto")

			if err != nil {
				go checkLogError(c.Request.URL.String(), "1", tpl.ExecuteTemplate(c.Writer, "error.html", "Error with uploaded picture, try again"))
			} else {
				pfi, err := companyLogo.Open()
				defer pfi.Close()

				if err != nil {
					go checkLogError(c.Request.URL.String(), "2", tpl.ExecuteTemplate(c.Writer, "error.html", "Error with uploaded picture, try again"))
				} else {
					company := c.PostForm("companyName")

					if company == "" {
						go checkLogError(c.Request.URL.String(), "3", tpl.ExecuteTemplate(c.Writer, "error.html", "Company name cannot be empty"))
					} else {
						fi, err := os.Open(path.Join("www/lib/imgs/companyImages", path.Base(company)))
						defer fi.Close()

						if err != nil && err != os.ErrExist {
							go checkLogError(c.Request.URL.String(), "4", os.Remove(path.Join("www/lib/imgs/companyImages", path.Base(company))))
							go checkLogError(c.Request.URL.String(), "5", tpl.ExecuteTemplate(c.Writer, "error.html", "An error occured, please try again"))
						} else {
							if err != os.ErrExist {
								writer := bufio.NewWriter(fi)
							}
							startDateString := c.PostForm("startDate")
							if err != nil {
								checkLogError(c.Request.URL.String(), "6", tpl.ExecuteTemplate(c.Writer, "error.html", "startTime name cannot be empty"))
							} else {
								startDate, err := time.Parse("Unix", startDateString)
								if err != nil {
								}
								jobTitle := c.PostForm("position")
								description := c.PostForm("description")
								location := c.PostForm("location")
								pay := c.PostForm("pay")
								expirationOfPosting := c.PostForm("expirationDate")
								contactInfo := c.PostForm("contactInfo")
								majors := c.PostForm("majors")
								typeOfProgram := c.PostForm("typeOfProgram")

								endDate := c.PostForm("dateEnd")
								tags := c.PostForm("tags")

								_, err := db.Exec(
									`INSERT INTO currentProgarms (
						company, companyLogo, jobTitle, description, location, pay, expirationOfPosting,
						contactInfo, majors, typeOfProgram, startDate, endDate
					) values (
						$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12
					)`, company, companyLogoLocation, jobTitle, description, location, pay, expirationOfPosting,
									contactInfo, majors, typeOfProgram, startDate, endDate)
								checkLogError(c.Request.URL.String(), "1", err)
							}
						}
					}
				}
			}
		})
	})

	r.POST("/admin/delete_program/:id", func(c *gin.Context) {
		checkAuth(c, func(c *gin.Context) {
			//get data from the request
			id := c.Param("id")

			_, err := db.Exec("DELETE FROM currentPrograms where id=$1", id)
			checkLogError(c.Request.URL.String(), "2", err)
		})
	})

	r.POST("/search", func(c *gin.Context) {
		checkAuth(c, func(c *gin.Context) {
			//get data from the request
			payString := c.PostForm("pay")
			pay, err := strconv.ParseFloat(payString, 32)
			checkLogError(c.Request.URL.String(), "1", err)
			if err == nil {
			}

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

func updateProgram(upProgram program, keyword, newValue string) error {
	var location = "updateProgram String"
	var err error

	_, err = db.Exec("UPDATE currentPrograms SET $1=$2 WHERE id=$3", keyword, newValue, upProgram.id)
	checkLogError(location, "updating a program in currentPrograms", err)

	return err
}

func updatePay(upProgram program, keyword string, newValue float32) error {
	var location = "updateProgram String"
	var err error

	_, err = db.Exec("UPDATE currentPrograms SET $1=$2 WHERE id=$3", keyword, newValue, upProgram.id)
	checkLogError(location, "updating a program in currentPrograms", err)

	return err
}

package main

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	//_ "github.com/mattn/go-sqlite3"
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
	tpl = template.Must(template.New("").ParseGlob("www/*.gohtml"))
	tpl = template.Must(tpl.ParseGlob("www/templates/*.gohtml"))

	//Route for the static files in www/
	r.Use(static.Serve("/www", static.LocalFile("www/", true)))

	r.GET("/", func(c *gin.Context) {
		tpl.ExecuteTemplate(c.Writer, "index.gohtml", nil)
	})

	r.GET("/about", func(c *gin.Context) {
		tpl.ExecuteTemplate(c.Writer, "about.gohtml", nil)
	})

	r.GET("/admin/add_program", func(c *gin.Context) {
		checkAuth(c, func(c *gin.Context) {
			tpl.ExecuteTemplate(c.Writer, "newProgram.gohtml", nil)
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
				go checkLogError(c.Request.URL.String(), "1", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Error with uploaded picture, try again"))
			} else {
				pfi, err := companyLogo.Open()
				defer pfi.Close()

				if err != nil {
					go checkLogError(c.Request.URL.String(), "2", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Error with uploaded picture, try again"))
				} else {
					buf := make([]byte, 512)
					_, err = pfi.Read(buf)

					if err != nil {
						go checkLogError(c.Request.URL.String(), "3", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Error with uploaded file, please try again"))
					} else {
						filetype := http.DetectContentType(buf)
						var extension string

						switch filetype {
						case "image/jpeg", "image/jpg":
							extension = ".jpeg"

						case "image/gif":
							extension = ".gif"

						case "image/png":
							extension = ".png"

						default:
							err = errors.New("Invalid file type uploaded")
						}

						if err != nil {
							go checkLogError(c.Request.URL.String(), "14", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Invalid file type for image file"))
						} else {
							company := c.PostForm("companyName")
							if company == "" {
								go checkLogError(c.Request.URL.String(), "15", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Company name cannot be empty"))
							} else {
								companyLogoLocation := path.Join("www/lib/imgs/companyImages", path.Base(company)+extension)
								fi, err := os.OpenFile(companyLogoLocation, os.O_CREATE, 0666)
								defer fi.Close()

								if err != nil && err != os.ErrExist {
									go checkLogError(c.Request.URL.String(), "4", os.Remove(companyLogoLocation))
									go checkLogError(c.Request.URL.String(), "5", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "An error occured, please try again"))
								} else {
									if err != os.ErrExist {
										err = nil
										var n int
										fi.Write(buf[:512])
										for {
											n, err = pfi.Read(buf)
											if err != nil && err != io.EOF {
												go checkLogError(c.Request.URL.String(), "15", os.Remove(companyLogoLocation))
												break
											}

											if n == 0 {
												break
											}

											if _, err := fi.Write(buf[:n]); err != nil {
												go checkLogError(c.Request.URL.String(), "16", os.Remove(companyLogoLocation))
												break
											}
										}
									} else {
										err = nil
									}

									if err != nil {
										go checkLogError(c.Request.URL.String(), "17", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "An error occured with the logo image, please try again"))
									} else {
										startDateString := c.PostForm("startDate")

										if startDateString == "" {
											checkLogError(c.Request.URL.String(), "6", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "startTime name cannot be empty"))
										} else {
											var startDate time.Time
											startDate, err = time.Parse("2006-01-02", startDateString)

											if err != nil {
												logError(c.Request.URL.String(), "21", err)
												checkLogError(c.Request.URL.String(), "7", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "An error occured, please try again"))
											} else {
												endDateString := c.PostForm("endDate")

												if endDateString == "" {
													checkLogError(c.Request.URL.String(), "8", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "End time cannot be empty"))
												} else {
													var endDate time.Time
													endDate, err = time.Parse("2006-01-02", endDateString)

													if err != nil {
														logError(c.Request.URL.String(), "22", err)
														checkLogError(c.Request.URL.String(), "9", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "An error occured, please try again"))
													} else {
														if startDate.After(endDate) {
															checkLogError(c.Request.URL.String(), "10", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Error, start date is after end date"))
														} else {
															if startDate.After(time.Now().AddDate(1, 0, 0)) {
																checkLogError(c.Request.URL.String(), "11", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Error, the start date is after a year from now, it must be within a year from the current day"))
															} else {
																if endDate.After(time.Now().AddDate(1, 0, 0)) {
																	checkLogError(c.Request.URL.String(), "12", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Error, the end date is after a year from now, it must be within a year from the current day"))
																} else {
																	expirationOfPostingString := c.PostForm("expirationDate")

																	if expirationOfPostingString == "" {
																		checkLogError(c.Request.URL.String(), "18", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Expiration time cannot be empty"))
																	} else {
																		var expirationDate time.Time
																		expirationDate, err = time.Parse("2006-01-02", expirationOfPostingString)
																		if err != nil {
																			logError(c.Request.URL.String(), "21", err)
																			checkLogError(c.Request.URL.String(), "22", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "An error occured, please try again"))
																		} else {
																			if expirationDate.After(time.Now().AddDate(1, 0, 0)) {
																				checkLogError(c.Request.URL.String(), "20", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Error, the expiration date is after a year from now, it must be within a year from the current day"))
																			} else {
																				if err != nil {
																					checkLogError(c.Request.URL.String(), "19", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "An error occured, please try again"))
																				} else {
																					jobTitle := c.PostForm("position")
																					description := c.PostForm("description")
																					location := c.PostForm("location")
																					pay := c.PostForm("pay")
																					contactInfo := c.PostForm("contactInfo")
																					typeOfProgram := c.PostForm("typeOfProgram")

																					endDate := c.PostForm("dateEnd")
																					majors := correctMajors(c.PostForm("tags"))

																					_, err = db.Exec(
																						`INSERT INTO currentProgarms (
																						company, companyLogo, jobTitle, description, location, pay, expirationOfPosting,
																						contactInfo, majors, typeOfProgram, startDate, endDate
																					) values (
																						$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12
																					)`, company, companyLogoLocation, jobTitle, description, location, pay, expirationDate,
																						contactInfo, majors, typeOfProgram, startDate, endDate,
																					)

																					if err != nil {
																						go checkLogError(c.Request.URL.String(), "13", tpl.ExecuteTemplate(c.Writer, "error.gohtml", "Error adding that item, please try again"))
																					} else {
																						c.Redirect(303, "/admin/add_program")
																					}
																				}
																			}
																		}
																	}
																}
															}
														}
													}
												}
											}
										}
									}
								}
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
			var conditionList []string
			conditionList = append(conditionList, "SELECT * FROM currentPrograms WHERE")
			payString := c.PostForm("pay")
			pay, err := strconv.ParseFloat(payString, 32)
			checkLogError(c.Request.URL.String(), "1", err)
			if err != nil {
				checkLogError(c.Request.URL.String(), "2", err)
			} else {
				if pay != -1 {
					conditionList = append(conditionList, "pay=$1", payString)
				}
			}

		})
	})

	r.GET("/login", func(c *gin.Context) {
		if isActiveSession(c.Request) {
			tpl.ExecuteTemplate(c.Writer, "error.gohtml", "You are already logged in!")
		} else {
			tpl.ExecuteTemplate(c.Writer, "login.gohtml", nil)
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
			go checkLogError(c.Request.URL.String(), "1", tpl.ExecuteTemplate(c.Writer, "login.gohtml", "BAD LOGIN!"))
		} else {
			if err != nil {
				go logError(c.Request.URL.String(), "2", err)
				go checkLogError(c.Request.URL.String(), "3", tpl.ExecuteTemplate(c.Writer, "login.gohtml", "ERROR LOGGING IN!"))
			} else {
				if checkPasswordHash(password, spassword) {
					uid := getUUID()
					http.SetCookie(c.Writer, &http.Cookie{Name: "uuid", Value: uid})

					_, err = db.Exec("INSERT INTO USER_SESSIONS (pid, uuid) VALUES ($1, $2)", user, uid)

					if err != nil {
						go logError(c.Request.URL.String(), "4", err)
						go checkLogError(c.Request.URL.String(), "5", tpl.ExecuteTemplate(c.Writer, "login.gohtml", "ERROR LOGGING IN!"))
					} else {
						c.Redirect(303, "/")
					}
				} else {
					go checkLogError(c.Request.URL.String(), "7", tpl.ExecuteTemplate(c.Writer, "login.gohtml", "BAD LOGIN!"))
				}
			}
		}
	})

	panic(r.Run(":6600"))
}

func correctMajors(majorsString string) string {
	var correctMajorSlice []string
	seenMajor := make(map[string]bool)
	for _, major := range strings.Split(majorsString, ",") {
		if _, exist := seenMajor[major]; !exist {
			correctMajorSlice = append(correctMajorSlice, major)
			seenMajor[major] = true
		}
	}
	sort.Strings(correctMajorSlice)
	return strings.Join(correctMajorSlice, ",")
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
	f, err := os.OpenFile("sdpass.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	for {
		select {
		case lErr = <-errorChannel:
			fmt.Println(lErr.Location, lErr.Sublocation, lErr)
			f.WriteString(fmt.Sprintf("%s, %s, %s\n", lErr.Location, lErr.Sublocation, lErr))
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

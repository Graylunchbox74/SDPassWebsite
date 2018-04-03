package main

import (
	"net/http"
	"strings"
)

func sayHello(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Path
	message = strings.TrimPrefix(message, "/")
	message = "Hello " + message

	w.Write([]byte(message))
}

func main() {
	http.Handle("/", http.FileServer(http.Dir("./www")))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
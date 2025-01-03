package main

import (
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"time"
)

type PageData struct {
	PageTitle string
	Text      string
}

func randomSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func createRedirect(text string) {
	test := randomSeq(8)
	fmt.Println("Redirecting '", text, "' to '", test)
}

func main() {
	fmt.Println("Start the shortener!")

	// init seed
	rand.New(rand.NewSource(time.Now().UnixNano()))

	templ := template.Must(template.ParseFiles("layout.html"))

	// handle request to the root url
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			PageTitle: "Simple URL shortener",
			Text:      "Type in a URL...",
		}
		templ.Execute(w, data)
	})

	// handle url submissions
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {

			// get text
			r.ParseForm()
			text := r.FormValue("textfield")

			createRedirect(text)

			// Create a PageData struct with the text entered
			pageData := PageData{
				PageTitle: "Simple URI shortener",
				Text:      text,
			}

			templ, err := template.ParseFiles("layout.html")
			if err != nil {
				return
			}
			templ.Execute(w, pageData)
		}
	})

	http.ListenAndServe(":80", nil)
}

package main

import (
	"context"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

func createRedirect(input_url string, collection *mongo.Collection, ctx context.Context) {
	// create random sequence of letters of length 8
	short := randomSeq(8)

	_, err := collection.InsertOne(ctx, bson.D{
		{Key: "original_url", Value: input_url},
		{Key: "short_url", Value: short},
	})

	if err != nil {
		fmt.Println("Failed to create redirect:", err)
	} else {
		fmt.Printf("Redirecting '%s' to '%s'\n", input_url, short)
	}
}

func getRedirect(shortCode string, collection *mongo.Collection, ctx context.Context) {
	fmt.Println("trying to get original_url from db:", shortCode)
}

func main() {
	fmt.Println("Starting the shortener!")

	url := os.Getenv("MONGO_URL")

	// Create a new client and connect to MongoDB
	client, err := mongo.Connect(options.Client().ApplyURI(url).SetMaxPoolSize(100))
	if err != nil {
		fmt.Printf("Failed to create MongoDB client: %v", err)
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Ping MongoDB to ensure the connection is established
	if err := client.Ping(context.Background(), nil); err != nil {
		fmt.Printf("Failed to ping MongoDB: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Connected to MongoDB!")

	// Select database and collection
	collection := client.Database("db").Collection("redirects")

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
			input_url := r.FormValue("textfield")

			createRedirect(input_url, collection, ctx)

			// Create a PageData struct with the text entered
			pageData := PageData{
				PageTitle: "Simple URI shortener",
				Text:      input_url,
			}

			templ, err := template.ParseFiles("layout.html")
			if err != nil {
				return
			}
			templ.Execute(w, pageData)
		}
	})

	http.HandleFunc("/short/", func(w http.ResponseWriter, r *http.Request) {
		// pattern to get the short url
		shortPattern := regexp.MustCompile(`.*/([a-zA-Z0-9]{8})$`)
		matches := shortPattern.FindStringSubmatch(r.URL.Path)
		shortCode := matches[1]

		getRedirect(shortCode, collection, ctx)
	})

	http.ListenAndServe(":80", nil)
}

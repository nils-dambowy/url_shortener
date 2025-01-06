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
	ShortURL  string
}

var result struct {
	OriginalURL string `bson:"original_url"`
}

func randomSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func createRedirect(input_url string, collection *mongo.Collection, ctx context.Context) string {
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
	return short
}

func getRedirect(shortCode string, collection *mongo.Collection, ctx context.Context) string {
	fmt.Println("Trying to get original_url from DB for shortCode:", shortCode)

	// Filter for the query
	filter := bson.D{{Key: "short_url", Value: shortCode}}

	// Projection to get only the original_url
	projection := options.FindOne().SetProjection(bson.D{
		{Key: "original_url", Value: 1},
		{Key: "_id", Value: 0},
	})

	// Perform the query
	query := collection.FindOne(ctx, filter, projection)

	// Log the query result object itself
	fmt.Println("Query result:", query)

	// Decode the result
	err := query.Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			fmt.Println("No document found with the given short_url:", shortCode)
		} else {
			fmt.Println("Error while fetching the document:", err)
		}
		return "" // Return an empty string if no result is found or an error occurs
	}

	// Successfully fetched the document, log the original_url
	fmt.Println("Original URL found:", result.OriginalURL)

	// Check if the URL already has a protocol (http:// or https://), otherwise add it
	if !regexp.MustCompile(`^https?://`).MatchString(result.OriginalURL) {
		result.OriginalURL = "http://" + result.OriginalURL
	}

	return result.OriginalURL
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
	// ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	// defer cancel()
	ctx := context.Background()

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

	templ := template.Must(template.ParseFiles("new_layout.html"))

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

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

			short := createRedirect(input_url, collection, ctx)

			// Create a PageData struct with the text entered
			pageData := PageData{
				PageTitle: "Simple URI shortener",
				Text:      input_url,
				ShortURL:  short,
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

		newURL := getRedirect(shortCode, collection, ctx)

		http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
	})

	http.ListenAndServe(":80", nil)
}

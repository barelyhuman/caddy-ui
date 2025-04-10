package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/barelyhuman/caddy-ui/migrate"
	"github.com/barelyhuman/caddy-ui/views"

	_ "github.com/mattn/go-sqlite3"

	"github.com/barelyhuman/go/env"
)

func getCaddyURL(path string) (string, error) {
	baseURL := env.Get("CADDY_URL", "http://localhost:2019")
	return url.JoinPath(baseURL, path)
}

func configEditorHandler(w http.ResponseWriter, r *http.Request) {
	// Set the Content-Type header to "text/html"
	w.Header().Set("Content-Type", "text/html")
	// HTML response with a basic code editor implemented using a textarea and JavaScript's fetch API
	views.Render(w, "ConfigEditor", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Set the Content-Type header to "text/html"
	w.Header().Set("Content-Type", "text/html")
	// HTML response with a basic code editor implemented using a textarea and JavaScript's fetch API
	views.Render(w, "Home", nil)
}

type ResponseError struct {
	err     error
	Message string `json:"error"`
}

func (e ResponseError) toJSONString() (string, error) {
	e.Message = e.err.Error()
	bytes, err := json.Marshal(e)
	return string(bytes), err
}

func fetchConfigHandler(w http.ResponseWriter, r *http.Request) {
	url, err := getCaddyURL("/config")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json, _ := ResponseError{
			err: err,
		}.toJSONString()

		io.WriteString(w, json)
		return
	}
	fmt.Printf("url: %v\n", url)
	resp, err := http.Get(url)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json, _ := ResponseError{
			err: err,
		}.toJSONString()
		io.WriteString(w, json)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println("Error copying response:", err)
	}
}

func uploadConfigHandler(w http.ResponseWriter, r *http.Request) {

	// Ensure the method is POST
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, `{"error": "Method not allowed. Expected POST"}`)
		return
	}

	// Read the config from the request body
	configBytes, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error": "Failed to read config: `+err.Error()+`"}`)
		return
	}

	url, err := getCaddyURL("/load")
	if err != nil {
		http.Error(w, "Error creating config url: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Upload the config to the backend service
	resp, err := http.Post(url, "application/json", bytes.NewReader(configBytes))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error": "Error uploading config: `+err.Error()+`"}`)
		return
	}
	defer resp.Body.Close()

	// Relay the response from the backend service
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println("Error copying response:", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	io.WriteString(w, `{"message": "Uploaded"}`)
}

func main() {
	db, err := sql.Open("sqlite3", "./data.sqlite3")
	if err != nil {
		log.Fatal("Failed to open database")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to database")
	}

	migrate.MigrateUp(db, "./migrate")

	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/config-editor", configEditorHandler)
	mux.HandleFunc("/fetch-config", fetchConfigHandler)
	mux.HandleFunc("/upload-config", uploadConfigHandler)

	log.Println("Listening on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

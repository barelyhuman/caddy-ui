package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/barelyhuman/go/env"
)

func getCaddyURL(path string) (string, error) {
	baseURL := env.Get("CADDY_URL", "http://localhost:2019")
	return url.JoinPath(baseURL, path)
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Set the Content-Type header to "text/html"
	w.Header().Set("Content-Type", "text/html")
	// HTML response with a basic code editor implemented using a textarea and JavaScript's fetch API
	html := `<html>
    <head>
        <link rel="stylesheet" href="https://rsms.me/raster/raster2.css?v=20">
        <style>
            body {
                margin: 20px;
                font-family: Arial, sans-serif;
            }
            #editor {
                width: 100%;
                height: 400px;
                border: 1px solid #ccc;
                font-family: monospace;
                font-size: 14px;
                padding: 10px;
                box-sizing: border-box;
            }
            button {
                margin: 5px;
                padding: 10px 20px;
            }
        </style>
    </head>
    <body>
        <h1>Config Editor</h1>
        <textarea id="editor"></textarea>
        <br/>
        <button onclick="fetchConfig()">Fetch Config</button>
        <button onclick="uploadConfig()">Save Config</button>
        <script>
            // Automatically fetch config on page load
            document.addEventListener('DOMContentLoaded', fetchConfig);

            function fetchConfig() {
                fetch('/fetch-config')
                    .then(async response => {
                        try{
                            if(!response.ok){
                                const res = await response.json()
                                 if (res.error) {
                                 alert(res.error)
                    }
                                return {}
                            }
                            return await response.json()
                        }catch(err){
                            return {}
                        }
                    })
                    .then(data => {
                        document.getElementById('editor').value = JSON.stringify(data, null, 2);
                    })
                    .catch(err => console.error('Error fetching config:', err));
            }

            function uploadConfig() {
                const configText = document.getElementById('editor').value;

                fetch('/upload-config', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: configText
                })
                .then(response => response.json())
                .then(data => {
                    if (data.error) {
                        alert(data.error);
                    } else {
                        alert('Config saved successfully!');
                    }
                })
                .catch(err => alert('Error uploading config: ' + err));
            }
        </script>
    </body>
</html>`
	_, err := w.Write([]byte(html))
	if err != nil {
		log.Println("Error writing response:", err)
	}
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
	http.HandleFunc("/", handler)
	http.HandleFunc("/fetch-config", fetchConfigHandler)
	http.HandleFunc("/upload-config", uploadConfigHandler)

	log.Println("Listening on :8081")
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

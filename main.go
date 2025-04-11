package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/barelyhuman/caddy-ui/data"
	"github.com/barelyhuman/caddy-ui/data/models/app_ports"
	"github.com/barelyhuman/caddy-ui/data/models/apps"
	"github.com/barelyhuman/caddy-ui/data/models/domains"
	"github.com/barelyhuman/caddy-ui/migrate"
	"github.com/barelyhuman/caddy-ui/views"
	"github.com/joho/godotenv"

	_ "github.com/mattn/go-sqlite3"

	"github.com/barelyhuman/go/env"
)

func getCaddyURL(path string) (string, error) {
	baseURL := env.Get("CADDY_URL", "http://localhost:2019")
	return url.JoinPath(baseURL, path)
}

func configEditorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	views.Render(w, "ConfigEditor", nil)
}

func SyncConfigForApp(appId int64) error {

	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	db, _ := data.GetDatabaseHandle()

	portRows, _ := db.Query(`select app_id, port from app_ports`)

	usedPortMap := map[int64]string{}

	for portRows.Next() {
		var appId int64
		var port string
		portRows.Scan(&appId, &port)
		if len(port) > 0 {
			usedPortMap[appId] = port
		}
	}

	if err := views.Render(w, "Home", struct {
		UsedPorts map[int64]string
	}{
		UsedPorts: usedPortMap,
	}); err != nil {
		fmt.Fprintf(w, "failed to render page, please try again later")
		log.Printf("failed with error: %v", err)
		return
	}
}

func appsHandler(w http.ResponseWriter, r *http.Request) {
	db, err := data.GetDatabaseHandle()
	if err != nil {
		log.Printf("failed with error: %v", err)
	}
	w.Header().Set("Content-Type", "text/html")
	data, err := apps.FindAll(db)
	if err != nil {
		log.Printf("failed with error: %v", err)
		// throw flash to request
	}

	if err = views.Render(w, "AppsHome", struct {
		Apps []apps.AppsWithIdentifier
	}{
		Apps: data,
	}); err != nil {
		fmt.Fprintf(w, "failed to render page, please try again later")
		log.Printf("failed with error: %v", err)
		return
	}
}

func appDetailsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html")

	id := r.PathValue("id")

	idInt, _ := strconv.Atoi(id)

	db, _ := data.GetDatabaseHandle()

	data, err := apps.FindById(db, int64(idInt))
	if err != nil {
		log.Println(err)
		return
	}
	ports, err := app_ports.FindByAppId(db, int64(idInt))
	if err != nil {
		log.Println(err)
		return
	}

	domainData, err := domains.FindByAppId(db, id)
	if err != nil {
		return
	}

	fmt.Printf("domainData: %v\n", domainData)

	views.Render(w, "AppsDetails", struct {
		App           apps.AppsWithIdentifier
		Ports         app_ports.AppPortsWithIdentifier
		PrimaryDomain domains.DomainsWithIdentifier
	}{
		App:           *data,
		Ports:         *ports,
		PrimaryDomain: *domainData,
	})
	return

}

func appDomainHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	r.ParseForm()

	domain := r.Form.Get("domain")

	id := r.PathValue("id")
	db, _ := data.GetDatabaseHandle()

	var existingDomainId int64
	rows, _ := db.Query(`select id from domains where app_id = ? limit 1`, id)
	for rows.Next() {
		rows.Scan(&existingDomainId)
	}
	rows.Close()

	if existingDomainId > 0 {
		_, err := db.Exec(`update domains set domain = ? where app_id = ?`, domain, id)
		if err != nil {
			log.Println("failed to update domain", err)
		}
	} else {
		_, err := db.Exec(`insert into domains (domain,app_id) values (?,?)`, domain, id)
		if err != nil {
			log.Println("failed to insert domain", err)
		}
	}

	http.Redirect(w, r, "/apps/"+id, http.StatusSeeOther)
}

func appDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		db, _ := data.GetDatabaseHandle()
		id := r.PathValue("id")
		idInt, _ := strconv.Atoi(id)

		rolledBack := false
		tx, _ := db.Begin()
		_, err := tx.Exec(`delete from apps where id = ?`, idInt)
		if err != nil {
			log.Println(err)
			tx.Rollback()
			rolledBack = true
		}
		_, err = tx.Exec(`delete from app_ports where app_id = ?`, idInt)
		if err != nil {
			log.Println(err)
			if !rolledBack {
				tx.Rollback()
				rolledBack = true
			}
		}

		if !rolledBack {
			tx.Commit()
		}

		http.Redirect(w, r, "/apps", http.StatusSeeOther)
		return
	}
}

func appsNewHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html")
		views.Render(w, "AppsCreate", nil)
		return
	}

	if r.Method == http.MethodPost {
		db, _ := data.GetDatabaseHandle()
		r.ParseForm()
		appName := r.Form.Get("name")
		appType := r.Form.Get("type")
		appPort := r.Form.Get("port")

		appInstance := apps.New()
		appInstance.Name = appName
		appInstance.InstanceID = 1
		appInstance.Type.Scan(appType)

		appRecord, err := appInstance.Save(db)

		if err != nil {
			log.Println(err)
			http.Redirect(w, r, "/apps/new", http.StatusSeeOther)
			return
		}

		if len(appPort) > 0 {
			port := app_ports.New()
			port.AppId = appRecord.ID
			port.Port = appPort
			_, err := port.Save(db)
			if err != nil {
				log.Println(err)
				http.Redirect(w, r, "/apps/new", http.StatusSeeOther)
				return
			}
		}

		http.Redirect(w, r, "/apps", http.StatusSeeOther)

		return
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
	godotenv.Load()

	db, err := data.GetDatabaseHandle()
	if err != nil {
		log.Fatalf("Failed to open database with error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to database")
	}

	migrate.MigrateUp(db, "./migrate")

	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)

	mux.HandleFunc("/apps", appsHandler)
	mux.HandleFunc("/apps/new", appsNewHandler)
	mux.HandleFunc("/apps/{id}", appDetailsHandler)
	mux.HandleFunc("/apps/{id}/delete", appDeleteHandler)
	mux.HandleFunc("/apps/{id}/domain", appDomainHandler)

	mux.HandleFunc("/config/editor", configEditorHandler)
	mux.HandleFunc("/fetch-config", fetchConfigHandler)
	mux.HandleFunc("/upload-config", uploadConfigHandler)

	log.Println("Listening on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/barelyhuman/caddy-ui/caddy"
	"github.com/barelyhuman/caddy-ui/data"
	"github.com/barelyhuman/caddy-ui/data/models/app_ports"
	"github.com/barelyhuman/caddy-ui/data/models/apps"
	"github.com/barelyhuman/caddy-ui/data/models/domains"
	"github.com/barelyhuman/caddy-ui/migrate"
	"github.com/barelyhuman/caddy-ui/views"
	"github.com/joho/godotenv"

	_ "github.com/mattn/go-sqlite3"
)

func configEditorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	views.Render(w, "ConfigEditor", nil)
}

func SyncConfigForApp(appId string) error {
	servers, err := caddy.GetServersConfig()
	if err != nil {
		return err
	}

	// Collect server keys that listen on :80 or :443.
	mappings := []string{}
	for key, server := range servers {
		for _, port := range server.Listen {
			if port == ":80" || port == ":443" {
				// Remove default file_server handler for :80.
				if port == ":80" {
					for i, route := range server.Routes {
						newHandles := []caddy.HandleDef{}
						for _, h := range route.Handle {
							// Skip the default file_server if no match criteria.
							if h.Handler == "file_server" && len(route.Match) == 0 {
								continue
							}
							newHandles = append(newHandles, h)
						}
						server.Routes[i].Handle = newHandles
					}
				}
				mappings = append(mappings, key)
				break
			}
		}
	}

	// Ensure at least one server listens on :443.
	has443 := false
	for _, srv := range servers {
		for _, port := range srv.Listen {
			if port == ":443" {
				has443 = true
				break
			}
		}
		if has443 {
			break
		}
	}
	if !has443 {
		key := "auto-443"
		servers[key] = caddy.Server{
			Listen: []string{":443"},
			Routes: []caddy.Route{},
		}
		mappings = append(mappings, key)
	}

	// Retrieve domain and port settings for the given app.
	db, err := data.GetDatabaseHandle()
	if err != nil {
		return err
	}
	domainData, err := domains.FindByAppId(db, appId)
	if err != nil {
		return err
	}
	appPort, err := app_ports.FindByAppId(db, appId)
	if err != nil {
		return err
	}
	dialAddr := "127.0.0.1:" + appPort.Port

	// For each mapping, update an existing reverse_proxy route or add a new one.
	for _, key := range mappings {
		server := servers[key]
		updated := false

		for i, route := range server.Routes {
			for j, handle := range route.Handle {
				if handle.Handler != "subroute" || len(handle.Routes) == 0 {
					continue
				}
				for k, subRoute := range handle.Routes {
					for l, subHandle := range subRoute.Handle {
						if subHandle.Handler == "reverse_proxy" && len(subHandle.Upstreams) > 0 {
							if subHandle.Upstreams[0].Dial != dialAddr {
								server.Routes[i].Handle[j].Routes[k].Handle[l].Upstreams[0].Dial = dialAddr
							}
							updated = true
							break
						}
					}
					if updated {
						break
					}
				}
				if updated {
					break
				}
			}
			if updated {
				break
			}
		}

		if !updated {
			newRoute := caddy.Route{
				Match: []caddy.Match{
					{Host: []string{domainData.Domain}},
				},
				Handle: []caddy.HandleDef{
					{
						Handler: "subroute",
						Routes: []caddy.Route{
							{
								Handle: []caddy.HandleDef{
									{
										Handler:   "reverse_proxy",
										Upstreams: []caddy.Upstream{{Dial: dialAddr}},
									},
								},
							},
						},
					},
				},
			}
			server.Routes = append(server.Routes, newRoute)
		}
		servers[key] = server
	}

	return caddy.SaveServersConfig(servers)
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

	db, _ := data.GetDatabaseHandle()

	data, err := apps.FindById(db, id)
	if err != nil {
		log.Println(err)
		return
	}
	ports, err := app_ports.FindByAppId(db, id)
	if err != nil {
		log.Println(err)
		return
	}

	domainData, err := domains.FindByAppId(db, id)
	if err != nil {
		return
	}

	views.Render(w, "AppsDetails", struct {
		App           apps.AppsWithIdentifier
		Ports         app_ports.AppPortsWithIdentifier
		PrimaryDomain domains.DomainsWithIdentifier
	}{
		App:           *data,
		Ports:         *ports,
		PrimaryDomain: *domainData,
	})
}

func syncConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	id := r.PathValue("id")

	SyncConfigForApp(id)

	jsonResponse, _ := ResponseJson{
		"message": "Done",
	}.toJSONString()

	io.WriteString(w, jsonResponse)
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

	SyncConfigForApp(id)

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
	config, err := caddy.GetFullConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json, _ := ResponseError{
			err: err,
		}.toJSONString()
		io.WriteString(w, json)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(config)
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
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error": "Failed to read config from request, make sure a valid config was sent"}`)
		return
	}
	defer r.Body.Close()
	err = caddy.SaveConfig(configBytes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		jsonReponse, _ := ResponseError{
			err: fmt.Errorf("failed to save config due to error: %v", err.Error()),
		}.toJSONString()
		io.WriteString(w, jsonReponse)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	jsonResponse, _ := ResponseJson{
		"message": "Config saved successfully",
	}.toJSONString()
	io.WriteString(w, jsonResponse)
}

type ResponseJson map[string]interface{}

func (e ResponseJson) toJSONString() (string, error) {
	bytes, err := json.Marshal(e)
	return string(bytes), err
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
	mux.HandleFunc("/apps/{id}/sync", syncConfigHandler)

	mux.HandleFunc("/config/editor", configEditorHandler)
	mux.HandleFunc("/fetch-config", fetchConfigHandler)
	mux.HandleFunc("/upload-config", uploadConfigHandler)

	allApps, _ := apps.FindAll(db)
	for _, v := range allApps {
		SyncConfigForApp(fmt.Sprintf("%v", v.ID))
	}

	log.Println("Listening on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

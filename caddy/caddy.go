package caddy

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

type AdminConfig struct {
	Disabled      bool     `json:"disabled,omitempty"`
	Listen        string   `json:"listen,omitempty"`
	EnforceOrigin bool     `json:"enforce_origin,omitempty"`
	Origins       []string `json:"origins,omitempty"`
}

// tiny subset of the caddy config to avoid bundling the entire
// caddy package as a dep
type Config struct {
	Admin *AdminConfig               `json:"admin,omitempty"`
	Apps  map[string]json.RawMessage `json:"apps,omitempty"`
}

func SaveConfig(fullConfig []byte) error {
	url, _ := getCaddyURL("/load")
	resp, err := http.Post(url, "application/json", bytes.NewReader(fullConfig))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusBadRequest {
		var errorMessage struct {
			Error string `json:"error"`
		}
		err := json.NewDecoder(resp.Body).Decode(&errorMessage)
		if err != nil {
			log.Println("error decoding", err)
		}
		fmt.Printf("errorMessage: %v\n", errorMessage)
		if len(errorMessage.Error) > 0 {
			return fmt.Errorf("%v", errorMessage.Error)
		}
	}
	return nil
}

func GetFullConfig() (Config, error) {
	var fullConfig Config
	url, err := getCaddyURL("/config/")
	if err != nil {
		return fullConfig, err
	}

	// Upload the config to the backend service
	resp, err := http.Get(url)
	if err != nil {
		return fullConfig, err
	}
	defer resp.Body.Close()

	buf, _ := io.ReadAll(resp.Body)
	json.Unmarshal(
		buf, &fullConfig,
	)

	return fullConfig, nil
}

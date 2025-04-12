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
	Admin *AdminConfig `json:"admin,omitempty"`
	Apps  *AppConfig   `json:"apps,omitempty"`
}

type ListenAddresses []string

type Upstream struct {
	Dial string `json:"dial"`
}

type HandleDef struct {
	Handler   string     `json:"handler,omitempty"`
	Upstreams []Upstream `json:"upstreams,omitempty"`
	Routes    []Route    `json:"routes,omitempty"`
}
type Match struct {
	Host []string `json:"host,omitempty"`
}

type Route struct {
	Handle   []HandleDef `json:"handle,omitempty"`
	Match    []Match     `json:"match,omitempty"`
	Terminal bool        `json:"terminal,omitempty"`
}

type Server struct {
	Listen ListenAddresses `json:"listen,omitempty"`
	Routes []Route         `json:"routes,omitempty"`
}

type AppConfig struct {
	HTTP struct {
		Servers ServersConfig `json:"servers,omitempty"`
	} `json:"http,omitempty"`
}

type ServersConfig map[string]Server

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

func SaveServersConfig(config ServersConfig) error {
	result, err := url.JoinPath("/config/apps/http/servers")
	if err != nil {
		return err
	}
	url, err := getCaddyURL(result)
	if err != nil {
		return err
	}
	value, _ := json.Marshal(config)
	resp, err := http.Post(url, "application/json", bytes.NewReader(
		value,
	))
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	v, _ := io.ReadAll(resp.Body)
	fmt.Printf("v: %v\n", string(v))

	return nil
}

func GetServersConfig() (ServersConfig, error) {
	var config ServersConfig

	result, err := url.JoinPath("/config/apps/http/servers")
	if err != nil {
		return config, err
	}
	url, err := getCaddyURL(result)
	if err != nil {
		return config, err
	}
	resp, err := http.Get(url)
	if err != nil {
		return config, err
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&config)
	return config, nil
}

func GetConfigAtPath(path string) (json.RawMessage, error) {
	var config json.RawMessage

	result, err := url.JoinPath("/config/", path)
	if err != nil {
		return config, err
	}
	url, err := getCaddyURL(result)
	if err != nil {
		return config, err
	}
	resp, err := http.Get(url)
	if err != nil {
		return config, err
	}
	defer resp.Body.Close()

	json.NewDecoder(resp.Body).Decode(&config)
	return config, nil
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

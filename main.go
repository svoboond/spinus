package main

import (
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

//go:embed all:config
var config embed.FS

//go:embed all:templates
var templates embed.FS

func getRoot(writer http.ResponseWriter, request *http.Request) {
	fmt.Printf("got / request\n")
	if request.URL.Path != "/" {
		http.NotFound(writer, request)
		return
	}
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", "POST")
		http.Error(
			writer,
			"That method is not allowed.",
			http.StatusMethodNotAllowed,
		)
		return
	}
	tmpl, err := template.ParseFS(
		templates,
		"templates/index.tmpl",
		"templates/header.tmpl",
		"templates/footer.tmpl",
	)
	if err != nil {
		fmt.Printf("error reading templates: %s\n", err)
		http.Error(
			writer,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}
	err = tmpl.Execute(writer, "Hello!")
	if err != nil {
		fmt.Printf("error executing templates: %s\n", err)
		http.Error(
			writer,
			"Internal Server Error",
			http.StatusInternalServerError,
		)
		return
	}
}

type Conf struct {
	Service struct {
		Port uint16 `yaml:"port"`
	} `yaml:"service"`
	Database struct {
		Host     string `yaml:"host"`
		Port     uint16 `yaml:"port"`
		Name     string `yaml:"name"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"database"`
}

func (c *Conf) Load() error {
	// declare two maps to hold yaml content
	base := make(map[string]any)
	local := make(map[string]any)
	// read base yaml file
	baseData, err := config.ReadFile("config/base.yaml")
	if err != nil {
		return fmt.Errorf("error opening base config file: %w", err)
	}
	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return fmt.Errorf(
			"error unmarshalling base config file: %w", err)
	}
	// read local yaml file
	localData, err := os.ReadFile("config/local.yaml")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Print("local config file does not exist\n")
		} else {
			return fmt.Errorf(
				"error opening local config file: %w", err)
		}
	}
	if err := yaml.Unmarshal(localData, &local); err != nil {
		return fmt.Errorf(
			"error unmarshalling local config file: %w", err)
	}
	// merge both yaml data recursively
	merged := mergeConfMaps(base, local)
	encoded, err := yaml.Marshal(merged)
	if err != nil {
		return fmt.Errorf("error marshalling merged config: %w", err)
	}
	err = yaml.Unmarshal(encoded, c)
	if err != nil {
		return fmt.Errorf("error unmarshalling merged config: %w", err)
	}
	return nil
}

func mergeConfMaps(s, d map[string]any) map[string]any {
	out := make(map[string]any, len(s))
	for sKey, sValue := range s {
		if sMap, ok := sValue.(map[string]any); ok {
			if dValue, ok := d[sKey]; ok {
				if dType, ok := dValue.(map[string]any); ok {
					out[sKey] = mergeConfMaps(sMap, dType)
					continue
				}
			}
		} else if dValue, ok := d[sKey]; ok {
			out[sKey] = dValue
			continue
		}
		out[sKey] = sValue
	}
	return out
}

func main() {
	conf := Conf{}
	err := conf.Load()
	if err != nil {
		fmt.Printf("error loading config: %s\n", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", getRoot)
	svcPort := conf.Service.Port
	fmt.Printf("listening on port %v\n", svcPort)
	err = http.ListenAndServe(fmt.Sprintf(":%v", svcPort), mux)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

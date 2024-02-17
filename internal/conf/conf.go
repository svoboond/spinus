package conf

import (
	"fmt"
	"os"

	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed base.yaml
var baseData []byte

type Conf struct {
	Service struct {
		Port uint16 `yaml:"port"`
	} `yaml:"service"`
	Postgres struct {
		Host     string `yaml:"host"`
		Port     uint16 `yaml:"port"`
		Name     string `yaml:"name"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"postgres"`
	Redis struct {
		Url string `yaml:"url"`
	} `yaml:"redis"`
	Log struct {
		Level   string `yaml:"level"`
		Handler string `yaml:"handler"`
	} `yaml:"log"`
}

func New(localPath string) (*Conf, error) {
	if localPath == "" {
		c := &Conf{}
		if err := yaml.Unmarshal(baseData, c); err != nil {
			return nil, fmt.Errorf("error unmarshalling base config: %w", err)
		}
		return c, nil
	}

	base := make(map[string]any)
	local := make(map[string]any)

	if err := yaml.Unmarshal(baseData, &base); err != nil {
		return nil, fmt.Errorf("error unmarshalling base config: %w", err)
	}

	localData, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("error opening local config file: %w", err)
	}
	if err := yaml.Unmarshal(localData, &local); err != nil {
		return nil, fmt.Errorf("error unmarshalling local config: %w", err)
	}

	merged := mergeConfMaps(base, local)
	encoded, err := yaml.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("error marshalling merged config: %w", err)
	}
	c := &Conf{}
	err = yaml.Unmarshal(encoded, c)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling merged config: %w", err)
	}
	return c, nil
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

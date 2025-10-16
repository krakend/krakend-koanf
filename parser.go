// Package koanf defines a config parser implementation based on the koanf pkg
package koanf

import (
	"fmt"
	"path/filepath"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/luraproject/lura/v2/config"
)

var delimiter = "."

// New creates a new parser using the koanf library
func New() Parser {
	return NewWithOptions(".")
}

func NewWithOptions(delimiter string) Parser {
	return Parser{koanf.New(delimiter)}
}

// Parser is a config parser using the viper library
type Parser struct {
	koanf *koanf.Koanf
}

// Parse reads and parses the configFile. The values of the file can be
// override with envvars, using the KRAKEND_ prefix
func (p Parser) Parse(configFile string) (config.ServiceConfig, error) {
	var cfg config.ServiceConfig

	var kp koanf.Parser
	ext := filepath.Ext(configFile)
	switch ext {
	case ".yaml":
	case ".yml":
		kp = yaml.Parser()
	case ".toml":
		kp = toml.Parser()
	default:
		kp = json.Parser()
	}

	if err := p.koanf.Load(file.Provider(configFile), kp); err != nil {
		return cfg, fmt.Errorf("'%s': %s", configFile, err.Error())
	}
	p.koanf.Load(env.Provider("KRAKEND_", delimiter, nil), nil)

	uCfg := koanf.UnmarshalConf{
		Tag: "mapstructure",
	}
	if err := p.koanf.UnmarshalWithConf("", &cfg, uCfg); err != nil {
		return cfg, fmt.Errorf("'%s': %s", configFile, err.Error())
	}

	cleanupServiceConfig(&cfg)
	if err := cfg.Init(); err != nil {
		return cfg, config.CheckErr(err, configFile)
	}

	return cfg, nil
}

// cleanupServiceConfig make sure ExtraConfig type is map[string]interface{}
func cleanupServiceConfig(cfg *config.ServiceConfig) {
	cfg.ExtraConfig = cleanConfigMap(cfg.ExtraConfig)
	for _, endpoint := range cfg.Endpoints {
		endpoint.ExtraConfig = cleanConfigMap(endpoint.ExtraConfig)

		for _, backend := range endpoint.Backend {
			backend.ExtraConfig = cleanConfigMap(backend.ExtraConfig)
		}
	}
}

func cleanConfigMap(cfg map[string]interface{}) map[string]interface{} {
	for k, v := range cfg {
		cfg[k] = cleanupMapValue(v)
	}
	return cfg
}

func cleanupMapValue(input interface{}) interface{} {
	switch data := input.(type) {
	case []interface{}:
		for key, value := range data {
			data[key] = cleanupMapValue(value)
		}
		return data
	case map[string]interface{}:
		for key, value := range data {
			data[key] = cleanupMapValue(value)
		}
		return data
	case map[interface{}]interface{}:
		output := make(map[string]interface{})
		for key, value := range data {
			output[fmt.Sprintf("%v", key)] = cleanupMapValue(value)
		}
		return output
	default:
		return data
	}
}

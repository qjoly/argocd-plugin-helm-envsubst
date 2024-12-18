package internal

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type HelmConfig struct {
	ArgocdConfig HexArgocdPluginConfig `yaml:"argocd,omitempty"`
}

type HexArgocdPluginConfig struct {
	ReleaseName           string   `yaml:"releaseName,omitempty"`
	Namespace             string   `yaml:"namespace,omitempty"`
	SkipCRD               bool     `yaml:"skipCRD,omitempty"`
	SyncOptionReplace     []string `yaml:"syncOptionReplace,omitempty"`
	ExternalHelmChartPath string   `yaml:"externalHelmChartPath,omitempty"` // relative path of the helm chart to use
}

func ReadChartYaml(path string) map[string]interface{} {
	var chartYaml map[string]interface{}

	bs, err := os.ReadFile(fmt.Sprintf("%s/Chart.yaml", path))
	if err != nil {
		log.Fatalf("Read Chart.yaml error: %v", err)
	}
	if err := yaml.Unmarshal(bs, &chartYaml); err != nil {
		log.Fatalf("Unmarshal Chart.yaml error: %v", err)
	}
	return chartYaml
}

func useExternalHelmChartPathIfSet() {

	log.Printf("Reading file: %s\n", os.Getenv("ARGOCD_APP_SOURCE_PATH"))
	bs, err := os.ReadFile(os.Getenv("ARGOCD_APP_SOURCE_PATH"))
	if err != nil {
		log.Fatalf("useExternalHelmChart - read values.yaml error : %v", err)
	}

	config := ReadArgocdConfig(string(bs))
	if len(config.ExternalHelmChartPath) > 0 {
		os.Chdir(config.ExternalHelmChartPath)
	}
}

func ReadArgocdConfig(configFile string) *HexArgocdPluginConfig {
	c := HelmConfig{}
	err := yaml.Unmarshal([]byte(configFile), &c)
	if err != nil {
		log.Fatalf("Unmarshal config file error: %v", err)
	}
	return &c.ArgocdConfig
}

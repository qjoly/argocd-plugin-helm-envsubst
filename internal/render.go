package internal

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

const (
	tokenPath     = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	caPath        = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	namespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

var (
	defaultDebugLogFilePath = "/tmp/argocd-helm-envsubst-plugin/"
	defaultHelmChartPath    = "."
	argocdEnvVarPrefix      = "ARGOCD_ENV"
)

type ConfigFileSeq struct {
	Seq  int
	Name string
}

type Renderer struct {
	debugLogFilePath string
}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (renderer *Renderer) RenderTemplate(helmChartPath string, debugLogFilePath string) {
	log.Println("Starting RenderTemplate")

	if len(debugLogFilePath) <= 0 {
		renderer.debugLogFilePath = defaultDebugLogFilePath
	} else {
		renderer.debugLogFilePath = debugLogFilePath
	}

	appRevision := os.Getenv("ARGOCD_APP_REVISION_SHORT")
	if len(appRevision) <= 0 {
		appRevision = "default-app-revision"
	}

	appName := os.Getenv("ARGOCD_APP_NAME")
	if len(appName) <= 0 {
		appName = "default-app-name"
	}

	tempDir := fmt.Sprintf("%s/%s-%s", os.TempDir(), appName, appRevision)

	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		log.Fatal("Temp dir not found, please check if the plugin is running in the correct order")
	}

	files, err := filepath.Glob(tempDir + "/*/build.yaml")

	if err != nil {
		log.Fatalf("Glob error: %v", err)
	}

	for _, file := range files {
		bs, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("Read file error: %v", err)
		}

		fmt.Println(string(bs))
	}

	// useExternalHelmChartPathIfSet()

	// configFileNames := renderer.FindHelmConfigs()
	// fmt.Printf("Found Helm config files: %v\n", configFileNames)

	// if len(configFileNames) > 0 {
	// 	for _, name := range configFileNames {
	// 		args = append(args, "-f")
	// 		args = append(args, name)
	// 		renderer.inlineEnvsubst(name, envs)
	// 		fmt.Printf("Processed config file: %s\n", name)
	// 	}
	// }

	// helmConfig := renderer.mergeYaml(configFileNames)
	// fmt.Printf("Merged Helm config: %s\n", helmConfig)

	// argocdConfig := ReadArgocdConfig(helmConfig)
	// fmt.Printf("ArgoCD config: %v\n", argocdConfig)

	// if len(argocdConfig.Namespace) > 0 {
	// 	args = append(args, "--namespace")
	// 	args = append(args, argocdConfig.Namespace)
	// 	fmt.Printf("Set namespace: %s\n", argocdConfig.Namespace)
	// }

	// if len(argocdConfig.ReleaseName) > 0 {
	// 	args = append(args, "--release-name")
	// 	args = append(args, argocdConfig.ReleaseName)
	// 	fmt.Printf("Set release name: %s\n", argocdConfig.ReleaseName)
	// }

	// if argocdConfig.SkipCRD {
	// 	args = append(args, "--skip-crds")
	// 	fmt.Println("Set to skip CRDs")
	// } else {
	// 	args = append(args, "--include-crds")
	// 	fmt.Println("Set to include CRDs")
	// }

	// if len(argocdConfig.SyncOptionReplace) > 0 {
	// 	postRendererScript := renderer.preparePostRenderer(argocdConfig.SyncOptionReplace)
	// 	args = append(args, "--post-renderer")
	// 	args = append(args, postRendererScript)
	// 	fmt.Printf("Set post-renderer script: %s\n", postRendererScript)
	// }

	// args = append(args, ".")
	// strCmd := strings.Join(args, " ")
	// fmt.Printf("Helm command: %s\n", strCmd)

	// cmd := exec.Command(command, strings.Split(strCmd, " ")...)
	// var out, stderr bytes.Buffer
	// cmd.Stdout = &out
	// cmd.Stderr = &stderr
	// err = cmd.Run()
	// if err != nil {
	// 	fmt.Printf("Exec helm template error: %s\n%s\n", err, stderr.String())
	// 	log.Fatalf("Exec helm template error: %s\n%s", err, stderr.String())
	// }

	// fmt.Println("Helm template executed successfully")
	// fmt.Println(out.String())
}

func (renderer *Renderer) envsubst(str string, envs []string) string {
	for _, env := range envs {
		envVar := os.Getenv(env)
		if len(envVar) > 0 {
			str = strings.Replace(str, "${"+env+"}", envVar, -1)
		}
	}
	return str
}

func (renderer *Renderer) preparePostRenderer(files []string) string {
	// Get the current temp path
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("osGetwd error: %s", err)
	}

	scriptPath := pwd + "/kustomize-renderer"
	kustomizeYamlPath := pwd + "/kustomization.yaml"
	allPath := pwd + "/all.yaml"

	// Create shell script
	script := fmt.Sprintf(`#!/bin/sh
	cat <&0 > %s
	kustomize build .`, allPath)

	err = os.WriteFile(scriptPath, []byte(script), 0777)
	if err != nil {
		log.Fatalf("Create kustomize-renderer error: %s", err)
	}

	// Create kustomize file
	kustomizations := []string{fmt.Sprintf(
		"resources:\n"+
			"- %s\n"+
			"patches:", allPath)}

	for _, file := range files {
		kustomizations = append(kustomizations, fmt.Sprintf(
			"- patch: |-\n"+
				"    - op: add\n"+
				"      path: /metadata/annotations/argocd.argoproj.io~1sync-options\n"+
				"      value: Replace=true\n"+
				"  target:\n"+
				"    name: %v", file))
	}

	err = os.WriteFile(kustomizeYamlPath, []byte(strings.Join(kustomizations, "\n")), 0777)
	if err != nil {
		log.Fatalf("Create %s error: %s", kustomizeYamlPath, err)
	}

	return scriptPath
}

func (renderer *Renderer) mergeYaml(configFiles []string) string {
	if len(configFiles) <= 0 {
		log.Fatalf("You must provide at least one config yaml")
	}
	var resultValues map[string]interface{}
	for _, filename := range configFiles {

		var override map[string]interface{}
		bs, err := os.ReadFile(filename)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(bs, &override); err != nil {
			continue
		}

		//check if is nil. This will only happen for the first filename
		if resultValues == nil {
			resultValues = override
		} else {
			for k, v := range override {
				resultValues[k] = v
			}
		}

	}
	bs, err := yaml.Marshal(resultValues)
	if err != nil {
		log.Fatalf("Marshal file error: %v", err)
	}

	return string(bs)
}

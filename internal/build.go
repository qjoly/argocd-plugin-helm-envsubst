package internal

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

var (
	defaultRepoConfigPath               = "/helm-working-dir/"
	defaultHelmRegistrySecretConfigPath = "/helm-working-dir/plugin-repositories/repositories.yaml"
	authHelmRegistry                    = []string{"https://gitlab.int.hextech.io"}
)

type HelmRepositoryConfig struct {
	ApiVersion   string       `default:"" yaml:"apiVersion"`
	Generated    string       `default:"0001-01-01T00:00:00Z" yaml:"generated"`
	Repositories []Repository `yaml:"repositories"`
}

type Repository struct {
	CaFile                string `default:"" yaml:"caFile"`
	CertFile              string `default:"" yaml:"certFile"`
	InsecureSkipTlsVerify bool   `default:"false" yaml:"insecure_skip_tls_verify"`
	KeyFile               string `default:"" yaml:"keyFile"`
	Name                  string `default:"" yaml:"name"`
	PassCredentialsAll    bool   `default:"false" yaml:"pass_credentials_all"`
	Username              string `default:"" yaml:"username"`
	Password              string `default:"" yaml:"password"`
	Url                   string `default:"" yaml:"url"`
}

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (builder *Builder) Build(helmChartPath string, repoConfigPath string, helmRegistrySecretConfigPath string) {
	log.Println("Starting Build process...")

	if len(helmChartPath) <= 0 {
		helmChartPath = defaultHelmChartPath
	}
	if len(repoConfigPath) <= 0 {
		repoConfigPath = defaultRepoConfigPath
	}
	if len(helmRegistrySecretConfigPath) <= 0 {
		helmRegistrySecretConfigPath = defaultHelmRegistrySecretConfigPath
	}

	appRevision := os.Getenv("ARGOCD_APP_REVISION_SHORT")
	if len(appRevision) <= 0 {
		appRevision = "default-app-revision"
	}

	appName := os.Getenv("ARGOCD_APP_NAME")
	if len(appName) <= 0 {
		appName = "default-app-name"
	}

	files, err := os.ReadDir(helmChartPath)
	if err != nil {
		log.Fatalf("Error reading directory: %v", err)
	}

	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Error getting current directory: %v", err)
	}

	log.Printf("Files in directory %s:", dir)
	for _, file := range files {
		log.Println(file.Name())
	}

	// GetAbsoluteDir
	absPath, err := filepath.Abs(helmChartPath)
	if err != nil {
		log.Fatalf("Error getting absolute path: %v", err)
	}

	log.Printf("Changing directory to: %s\n", absPath)
	err = os.Chdir(helmChartPath)
	if err != nil {
		log.Fatalf("Error changing directory: %v", err)
	}

	// Create a tempDir dedicated for the helm chart
	// We will untar the helm chart in this directory
	tempDir := fmt.Sprintf("%s/%s-%s", os.TempDir(), appName, appRevision)

	// If exists, remove the tempDir
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		err = os.RemoveAll(tempDir)
		if err != nil {
			log.Fatalf("Error removing temp directory: %v", err)
		}
	}

	err = os.Mkdir(tempDir, 0700)
	if err != nil {
		log.Fatalf("Error creating temp directory: %v", err)
	}

	log.Printf("Created temp directory: %s\n", tempDir)

	for _, file := range files {

		// Skip if file is a directory or not a yaml file
		if file.IsDir() || (!strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml")) {
			continue
		}

		fileContent, err := os.ReadFile(file.Name())
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}
		application := Application{}
		if err := yaml.Unmarshal(fileContent, &application); err != nil {
			log.Printf("Error unmarshal yaml: %v. Skipping...", err)
			continue
		}

		log.Println("Manifest name:", application.Metadata.Name)

		helmRegistry := application.Spec.Source.RepoURL
		if !strings.HasPrefix(helmRegistry, "https://") {
			log.Println("Helm registry is not https, skipping...")
			continue
		}

		sysCommand := "helm"
		sysArgs := []string{"pull", application.Spec.Source.Chart, "--repo", application.Spec.Source.RepoURL, "--untar", "--untardir", tempDir}
		sysCmd := exec.Command(sysCommand, sysArgs...)
		var out, stderr bytes.Buffer
		sysCmd.Stdout = &out
		sysCmd.Stderr = &stderr
		err = sysCmd.Run()
		if err != nil {
			log.Fatalf("Error running helm pull: %s\n%s", err, stderr.String())
		}
		err = cleanupTempDir(tempDir)
		if err != nil {
			log.Fatalf("Error cleaning up temp directory: %v", err)
		}

		chartPath := filepath.Join(tempDir, application.Spec.Source.Chart)
		if _, err := os.Stat(chartPath); os.IsNotExist(err) {
			log.Fatalf("Directory %s does not exist", chartPath)
		}

		chartYaml := ReadChartYaml(chartPath)

		isDependency := false
		dependencies := chartYaml["dependencies"]
		if dependencies == nil || len(dependencies.([]interface{})) <= 0 {
			log.Println("No dependencies found.")
		} else {
			isDependency = true
			log.Println("Dependencies found.")
		}

		if isDependency {
			log.Println("Generating repository config...")
			os.Chdir(tempDir + "/" + application.Spec.Source.Chart)
			sysArgs = []string{"dependency", "build", "--repository-config", repoConfigPath + application.Metadata.Name + ".yaml"}
			sysCmd = exec.Command(sysCommand, sysArgs...)
			sysCmd.Run()
		}

		overrideValuesPath := ""
		// if Values override is set, create a file override.values.yaml
		if application.Spec.Source.Helm.Values != "" {
			log.Println("Values file found, will use it to override values.")

			overrideValuesPath = fmt.Sprintf("%s/override.values.yaml", chartPath)
			err = os.WriteFile(overrideValuesPath, applyEnvOnValues([]byte(application.Spec.Source.Helm.Values)), 0600)
			if err != nil {
				log.Fatalf("Error writing override values: %v", err)
			}
		}

		sysArgs = []string{"template", application.Metadata.Name, chartPath}
		if application.Spec.Destination.Namespace == "" {
			sysArgs = append(sysArgs, "--namespace", os.Getenv("ARGOCD_APP_NAMESPACE"))
		} else {
			sysArgs = append(sysArgs, "--namespace", application.Spec.Destination.Namespace)
		}

		if overrideValuesPath != "" {
			sysArgs = append(sysArgs, "--values", overrideValuesPath)
		}

		sysCmd = exec.Command(sysCommand, sysArgs...)
		sysCmd.Stdout = &out
		sysCmd.Stderr = &stderr
		err = sysCmd.Run()
		if err != nil {
			log.Fatalf("Error running helm template: %s\n%s", err, stderr.String())
		}
		fmt.Println(out.String())

		buildPath := fmt.Sprintf("%s/%s/build.yaml", tempDir, application.Metadata.Name)
		err = os.WriteFile(buildPath, out.Bytes(), 0600)
		if err != nil {
			log.Fatalf("Error writing override values: %v", err)
		}
	}

	//	log.Fatal("stop here")

	// Use app name as config file name
	// repositoryConfigName := repoConfigPath + chartYaml["name"].(string) + ".yaml"
	// log.Printf("repositoryConfigName: %s\n", repositoryConfigName)

	// log.Println("Generating repository config...")
	// builder.generateRepositoryConfig(repositoryConfigName, chartYaml, helmRegistrySecretConfigPath)

	// log.Println("Executing helm dependency build...")
	// builder.executeHelmDependencyBuild(repositoryConfigName)

	// log.Println("Build process completed.")
}

func (builder *Builder) generateRepositoryConfig(repositoryConfigName string, chartYaml map[string]interface{}, helmRegistrySecretConfigPath string) {
	repos := []Repository{}
	// Read dependencies from Chart.yaml, and generate repositories.yaml from it
	for _, dep := range chartYaml["dependencies"].([]interface{}) {
		d := dep.(map[interface{}]interface{})
		repositoryUrl := d["repository"].(string)

		// Do not include repository url in the repositories.yaml if it is not https
		// Helm does not create an [app]-index.yaml that contains all the version of the chart for non-https repo
		// Including the url in the repositories.yaml will cause the helm to lookup for the index file and fail
		if !strings.HasPrefix(repositoryUrl, "https://") {
			continue
		}

		name := d["name"].(string)
		username := ""
		password := ""
		for _, authReg := range authHelmRegistry {
			if strings.HasPrefix(repositoryUrl, authReg) {
				// Read username password from /helm-working-dir/plugin-repositories/repositories.yaml
				u, p := builder.readRepositoryConfig(repositoryUrl, helmRegistrySecretConfigPath)
				username = u
				password = p
				break
			}
		}

		repos = append(repos, Repository{
			Name:     name,
			Url:      repositoryUrl,
			Username: username,
			Password: password,
		})
	}

	repoConfig := HelmRepositoryConfig{
		Generated:    "0001-01-01T00:00:00Z",
		Repositories: repos,
	}

	yamlConfig, err := yaml.Marshal(repoConfig)
	if err != nil {
		log.Fatalf("Marshal helm repository yaml error: %v", err)
	}
	fmt.Println(string(yamlConfig))

	err = os.WriteFile(repositoryConfigName, []byte(yamlConfig), 0777)
	if err != nil {
		log.Fatalf("Write helm repository yaml error: %v", err)
	}
}

func (builder *Builder) readRepositoryConfig(repositoryUrl string, helmRegistrySecretConfigPath string) (string, string) {
	repo := HelmRepositoryConfig{}

	// Read helm repository config created by Terraform
	bs, err := os.ReadFile(helmRegistrySecretConfigPath)
	if err != nil {
		log.Fatalf("Error reading repositories.yaml: %v", err)
	}

	if err := yaml.Unmarshal(bs, &repo); err != nil {
		log.Fatalf("Error unmarshal helmRegistrySecretConfigPath: %v", err)
	}

	// Return the username password if url matches
	for _, r := range repo.Repositories {
		if r.Url == repositoryUrl {
			return r.Username, r.Password
		}
	}
	return "", ""
}

func (builder *Builder) executeHelmDependencyBuild(repositoryConfigName string) {
	command := "helm"
	args := []string{"dependency", "build", "--repository-config", repositoryConfigName}
	cmd := exec.Command(command, args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Exec helm dependency build error: %s\n%s", err, stderr.String())
	}
	log.Println(out.String())
}

// When doing the helm pull, helm will create a directory with the chart name (e.g. cloudflare-tunnel)
// and another with the archive name (e.g. cloudflare-tunnel-0.3.2.tgz).
// To avoid to have this unnecessary directory, we will remove all the .tgz files in the tempDir
func cleanupTempDir(tempDir string) error {
	// in tempDir, remove all files that end with .tgz
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && strings.HasSuffix(info.Name(), ".tgz") {
			if err := os.Remove(path); err != nil {
				log.Printf("Error removing file %s: %v", path, err)
			} else {
				log.Printf("Removed file %s", path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

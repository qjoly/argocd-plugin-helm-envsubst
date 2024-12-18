package internal

type Metadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

type Helm struct {
	Values string `yaml:"values"`
}

type Source struct {
	RepoURL        string `yaml:"repoURL"`
	Chart          string `yaml:"chart"`
	TargetRevision string `yaml:"targetRevision"`
	Helm           Helm   `yaml:"helm"`
}

type Destination struct {
	Server    string `yaml:"server"`
	Namespace string `yaml:"namespace"`
}

type AutomatedSyncPolicy struct {
	Prune bool `yaml:"prune"`
}

type SyncPolicy struct {
	Automated   AutomatedSyncPolicy `yaml:"automated"`
	SyncOptions []string            `yaml:"syncOptions"`
}

type Spec struct {
	Project     string      `yaml:"project"`
	Source      Source      `yaml:"source"`
	Destination Destination `yaml:"destination"`
	SyncPolicy  SyncPolicy  `yaml:"syncPolicy"`
}

type Application struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

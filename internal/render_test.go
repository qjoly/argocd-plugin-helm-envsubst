package internal_test

import (
	"fmt"
	"os"
	"testing"
)

const (
	TEST_ARGOCD_ENV_ENVIRONMENT = "prod"
	TEST_ARGOCD_ENV_CLUSTER     = "blockchain"
)

func setup(t *testing.T) {
	if err := os.Mkdir("config", os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Create(fmt.Sprintf("config/%s_%s.yaml", TEST_ARGOCD_ENV_CLUSTER, TEST_ARGOCD_ENV_ENVIRONMENT)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Create(fmt.Sprintf("config/%s.yaml", TEST_ARGOCD_ENV_ENVIRONMENT)); err != nil {
		t.Fatal(err)
	}

	os.Setenv("ARGOCD_ENV_ENVIRONMENT", TEST_ARGOCD_ENV_ENVIRONMENT)
	os.Setenv("ARGOCD_ENV_CLUSTER", TEST_ARGOCD_ENV_CLUSTER)
}

func teardown(t *testing.T) {
	if err := os.RemoveAll("config"); err != nil {
		t.Fatal(err)
	}
}

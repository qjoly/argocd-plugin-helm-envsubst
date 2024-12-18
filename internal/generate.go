package internal

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (generator *Generator) Generate() {
	// Take stdin, substitute env vars and output the result

	// Check if stdin is empty
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		log.Fatal("stdin is empty")
	}

	content, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println("Error reading stdin:", err)
		return
	}
	fmt.Println(string(applyEnvOnValues(content)))
}

func applyEnvOnValues(values []byte) []byte {
	for _, env := range os.Environ() {
		// For security reason, we will skip all the env that start with ARGOCD_ and KUBERNETES_
		// These env are set by ArgoCD and Kubernetes and we don't want to expose them in any manifest
		if strings.HasPrefix(env, "ARGOCD_") || strings.HasPrefix(env, "KUBERNETES_") {
			log.Printf("Skippinp env: %s", env)
			continue
		}

		pair := strings.SplitN(env, "=", 2)
		values = bytes.ReplaceAll(values, []byte(fmt.Sprintf("#%s#", pair[0])), []byte(pair[1]))
	}

	return values
}

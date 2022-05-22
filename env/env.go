// Package env provides common configuration provided with environment
// variables.
package env

import (
	"fmt"
	"os"
)

// Env represents all config vars from the .env file.
type Env struct {
	Email            string
	CloudPassword    string
	ResourcePassword string
}

// verify verifies that all env vars are set.
func (e *Env) verify() error {
	if e.Email == "" {
		return fmt.Errorf("FHOME_EMAIL is not set")
	}

	if e.CloudPassword == "" {
		return fmt.Errorf("FHOME_CLOUD_PASSWORD is not set")
	}

	if e.ResourcePassword == "" {
		return fmt.Errorf("FHOME_RESOURCE_PASSWORD is not set")
	}

	return nil
}

// Load loads env vars from shell to e.
func (e *Env) Load() error {
	e.Email = os.Getenv("FHOME_EMAIL")
	e.CloudPassword = os.Getenv("FHOME_CLOUD_PASSWORD")
	e.ResourcePassword = os.Getenv("FHOME_RESOURCE_PASSWORD")

	return e.verify()
}

func (e Env) String() string {
	return fmt.Sprint("email:", e.Email, " password:", e.CloudPassword, "resourcePassword:", e.ResourcePassword)
}

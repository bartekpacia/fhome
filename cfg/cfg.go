// Package cfg provides common configuration provided with environment
// variables.
package cfg

import (
	"fmt"
	"os"
)

// Config for the tool.
type Config struct {
	Email            string
	CloudPassword    string
	ResourcePassword string
}

// Verify verifies that all env vars are set.
func (e *Config) Verify() error {
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
func (e *Config) Load() error {
	email, ok := os.LookupEnv("FHOME_EMAIL")
	if !ok {
		return fmt.Errorf("FHOME_EMAIL is not set")
	}
	e.Email = email

	cloudPassword, ok := os.LookupEnv("FHOME_CLOUD_PASSWORD")
	if !ok {
		return fmt.Errorf("FHOME_CLOUD_PASSWORD is not set")
	}
	e.CloudPassword = cloudPassword

	resourcePassword, ok := os.LookupEnv("FHOME_RESOURCE_PASSWORD")
	if !ok {
		return fmt.Errorf("FHOME_RESOURCE_PASSWORD is not set")
	}
	e.ResourcePassword = resourcePassword

	return e.Verify()
}

func (e Config) String() string {
	return fmt.Sprint("email:", e.Email, " password:", e.CloudPassword, "resourcePassword:", e.ResourcePassword)
}

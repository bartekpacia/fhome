// Package cfg provides common configuration provided with environment
// variables.
package cfg

import "fmt"

// Config required to authenticate to F&Home Cloud.
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

func (e Config) String() string {
	return fmt.Sprint(
		"email:", e.Email,
		"password:", e.CloudPassword,
		"resourcePassword:", e.ResourcePassword,
	)
}

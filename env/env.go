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

func (e Env) String() string {
	return fmt.Sprint("email:", e.Email, " password:", e.CloudPassword, "resourcePassword:", e.ResourcePassword)
}

// Load loads env vars from shell to e.
func (e *Env) Load() {
	e.Email = os.Getenv("FHOME_EMAIL")
	e.CloudPassword = os.Getenv("FHOME_CLOUD_PASSWORD")
	e.ResourcePassword = os.Getenv("FHOME_RESOURCE_PASSWORD")
}

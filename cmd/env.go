package main

import (
	"fmt"
	"os"
)

// Env represents all config vars from the .env file.
type Env struct {
	email    string
	cloudPassword string
	resourcePassword string
}

func (e Env) String() string {
	return fmt.Sprint("email:", e.email, " password:", e.cloudPassword, "resourcePassword:", e.resourcePassword)
}

// Load loads env vars from shell to e.
func (e *Env) Load() {
	e.email = os.Getenv("FHOME_EMAIL")
	e.cloudPassword = os.Getenv("FHOME_CLOUD_PASSWORD")
	e.resourcePassword = os.Getenv("FHOME_RESOURCE_PASSWORD")
}

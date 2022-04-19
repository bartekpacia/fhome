package main

import (
	"fmt"
	"os"
)

// Env represents all config vars from the .env file.
type Env struct {
	email        string
	password     string
	passwordHash string
}

func (e Env) String() string {
	return fmt.Sprint("listenHost:", e.email, " password:", e.password, " passwordHash:", e.passwordHash)
}

// Load loads env vars from shell to e.
func (e *Env) Load() {
	e.email = os.Getenv("FHOME_EMAIL")
	e.password = os.Getenv("FHOME_PASSWORD")
	e.passwordHash = os.Getenv("FHOME_PASSWORD_HASH")
}

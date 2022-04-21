package main

import (
	"fmt"
	"os"
)

// Env represents all config vars from the .env file.
type Env struct {
	email    string
	password string
}

func (e Env) String() string {
	return fmt.Sprint("email:", e.email, " password:", e.password)
}

// Load loads env vars from shell to e.
func (e *Env) Load() {
	e.email = os.Getenv("FHOME_EMAIL")
	e.password = os.Getenv("FHOME_PASSWORD")
}

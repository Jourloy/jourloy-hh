package main

import (
	"github.com/Jourloy/jourloy-hh/internal/server"
	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

func main() {
	log.SetLevel(log.DebugLevel)

	if err := godotenv.Load(`.env`); err != nil {
		log.Fatal(`Error loading .env file`)
	} else {
		log.Debug(`Loaded .env file`)
	}

	server.Start()
}

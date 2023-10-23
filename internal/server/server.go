package server

import (
	"os"
	"strconv"
	"time"

	"github.com/Jourloy/jourloy-hh/internal/server/handlers"
	"github.com/Jourloy/jourloy-hh/internal/server/storage"
	"github.com/charmbracelet/log"
	"github.com/gin-gonic/gin"
)

func Start() {
	host, exist := os.LookupEnv(`HOST`)
	if !exist {
		log.Fatal(`Error loading HOST environment variable`)
	}

	// Create storage
	database := storage.NewRepository()

	// Create a new router
	r := gin.New()

	// Use the logger
	r.Use(logger())

	// Groups
	authGroup := r.Group("/auth")

	// Register handlers
	handlers.RegisterAuthHandler(authGroup, database)

	log.Debug(`Server started`)

	if err := r.Run(host); err != nil {
		log.Fatal(err)
	}

}

func logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := log.NewWithOptions(os.Stderr, log.Options{Prefix: `[server]`, Level: log.DebugLevel})

		t := time.Now()
		c.Next()

		latency := time.Since(t)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path

		if method == `GET` {
			responseSize := c.Writer.Size()
			logger.Info(
				method,
				`status`,
				strconv.Itoa(status),
				`size`,
				responseSize,
				`path`,
				path,
				`latency`,
				latency,
			)
		} else {
			logger.Info(
				method,
				`status`,
				strconv.Itoa(status),
				`path`,
				path,
				`latency`,
				latency,
			)
		}
	}
}

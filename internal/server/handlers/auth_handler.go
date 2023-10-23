package handlers

import (
	"github.com/Jourloy/jourloy-hh/internal/server/auth"
	"github.com/Jourloy/jourloy-hh/internal/server/storage"
	"github.com/gin-gonic/gin"
)

func RegisterAuthHandler(g *gin.RouterGroup, d *storage.PostgresRepository) {
	authService := auth.NewAuthService(*d)

	g.GET(`/`, authService.Redirect)
	g.GET(`/callback`, authService.Callback)
}

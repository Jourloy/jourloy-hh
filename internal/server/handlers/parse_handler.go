package handlers

import (
	"github.com/Jourloy/jourloy-hh/internal/server/parser"
	"github.com/Jourloy/jourloy-hh/internal/server/storage"
	"github.com/gin-gonic/gin"
)

func RegisterParsehHandler(g *gin.RouterGroup, d *storage.PostgresRepository) {
	parseService := parser.NewParserService(*d)
	parseService.StartTickers()
}

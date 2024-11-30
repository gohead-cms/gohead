// cmd/main.go
package main

import (
	"github.com/gin-gonic/gin"
	ginlogrus "github.com/toorop/gin-logrus"
	"gitlab.com/sudo.bngz/gohead/internal/api/handlers"
	"gitlab.com/sudo.bngz/gohead/internal/api/middleware"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
	"gitlab.com/sudo.bngz/gohead/pkg/logger"
)

func main() {
	// Initialize the logger
	logger.InitLogger()

	// Initialize the database
	if err := database.InitDatabase(); err != nil {
		logger.Log.Fatal("Failed to connect to database!", err)
	}

	router := gin.New()
	router.Use(ginlogrus.Logger(logger.Log), gin.Recovery())

	// Public routes
	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", handlers.Register)
		authRoutes.POST("/login", handlers.Login)
	}

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/content-types", middleware.AuthorizeRole("admin"), handlers.CreateContentType)
		protected.Any("/:contentType", handlers.DynamicContentHandler)
		protected.Any("/:contentType/:id", handlers.DynamicContentHandler)
	}

	// Start the server
	logger.Log.Info("Starting server on port 8080")
	router.Run(":8080")
}

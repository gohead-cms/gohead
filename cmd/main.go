// cmd/main.go
package main

import (
	"github.com/gin-gonic/gin"
	"gitlab.com/sudo.bngz/gohead/internal/api/handlers"
	"gitlab.com/sudo.bngz/gohead/internal/api/middleware"
	"gitlab.com/sudo.bngz/gohead/pkg/database"
)

func main() {
	// Initialize the database
	if err := database.InitDatabase(); err != nil {
		panic("Failed to connect to database!")
	}

	router := gin.Default()

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
		// Content Types - Only Admins
		protected.POST("/content-types", middleware.AuthorizeRole("admin"), handlers.CreateContentType)

		// Content Items
		protected.Any("/:contentType", handlers.DynamicContentHandler)
		protected.Any("/:contentType/:id", handlers.DynamicContentHandler)
	}

	// Start the server
	router.Run(":8080")
}

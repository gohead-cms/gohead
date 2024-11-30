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
	router.Use(middleware.AuthMiddleware())

	// Route to create content types
	router.POST("/content-types", handlers.CreateContentType)

	// Dynamic route for content items
	router.Any("/:contentType", handlers.DynamicContentHandler)
	router.Any("/:contentType/:id", handlers.DynamicContentHandler)

	// Start the server
	router.Run(":8080")
}

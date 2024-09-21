package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Set up the HTML renderer
	r.LoadHTMLGlob("public/*.html")

	// Serve static files
	r.Static("/static", "public")

	// Define routes
	r.GET("/", serveHome)

	r.Run(":8080")
}

func serveHome(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

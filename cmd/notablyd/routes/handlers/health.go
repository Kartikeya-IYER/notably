package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func GetHealth(c *gin.Context) {
	message := fmt.Sprintf("Hello, the Notably web server is alive. The current time is %s",
		time.Now().UTC().Format(time.RFC3339))
	c.IndentedJSON(http.StatusOK, gin.H{"message:": message})
}

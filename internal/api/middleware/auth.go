package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// APIKeyAuth secures the endpoints against timing attacks
func APIKeyAuth() gin.HandlerFunc {
	// Cache the expected key on boot so we don't read the env file on every single request
	expectedKey := []byte("Bearer " + os.Getenv("INGEST_API_KEY"))

	return func(c *gin.Context) {
		authHeader := []byte(c.GetHeader("Authorization"))

		// ConstantTimeCompare ensures the CPU takes the exact same amount of time
		// to reject a bad password regardless of how many characters they guessed correctly.
		if subtle.ConstantTimeCompare(authHeader, expectedKey) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid or missing API key"})
			c.Abort()
			return
		}

		c.Next()
	}
}

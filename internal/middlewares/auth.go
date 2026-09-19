package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"hospital-middleware/internal/utils"
)

const StaffIDKey = "staff_id"
const HospitalKey = "hospital"

func Authenticate(jwtUtil *utils.JwtUtil) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the token from the Authorization header
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header"})
			c.Abort()
			return
		}

		// Parse and validate the token
		claims, err := jwtUtil.ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Store the staff ID and hospital in the context for downstream handlers
		c.Set(StaffIDKey, claims.StaffID)
		c.Set(HospitalKey, claims.Hospital)
		c.Next()
	}
}

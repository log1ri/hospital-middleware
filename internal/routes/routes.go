package routes

import (
	"github.com/gin-gonic/gin"
	"hospital-middleware/internal/handlers"
	"hospital-middleware/internal/middlewares"
	"hospital-middleware/internal/utils"
)

func SetupRoutes(r *gin.Engine, staffHandler *handlers.StaffHandler, patientHandler *handlers.PatientHandler, jwtUtil *utils.JwtUtil) {

	r.GET("/health", handlers.Health)

	staff := r.Group("/staff")
	staff.POST("/create", staffHandler.CreateStaff)
	staff.POST("/login", staffHandler.LoginStaff)

	r.GET("/patient/search", middlewares.Authenticate(jwtUtil), patientHandler.Search)

}

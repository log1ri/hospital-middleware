package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"hospital-middleware/internal/clients"
	"hospital-middleware/internal/utils"

	"hospital-middleware/internal/config"
	"hospital-middleware/internal/db"
	"hospital-middleware/internal/handlers"
	"hospital-middleware/internal/repositories"
	"hospital-middleware/internal/routes"
	"hospital-middleware/internal/services"
)

func main() {

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Configuration loaded successfully")

	// Initialize the database
	database, err := db.Init(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Database initialized successfully")

	// Set up the Gin router and routes
	r := gin.Default()
	log.Println("Gin router initialized successfully")

	// Initialize services and handlers
	staffRepo := &repositories.StaffRepository{DB: database}
	jwtUtil := &utils.JwtUtil{
		SecretKey:  []byte(cfg.JWTSecret),
		Expiration: cfg.JWTExpiration,
	}

	staffService := &services.StaffService{
		Repo: staffRepo,
		JWT:  jwtUtil,
	}

	staffHandler := &handlers.StaffHandler{Service: staffService}

	patientService := &services.PatientService{
		Repo:      &repositories.PatientRepository{DB: database},
		HospitalA: &clients.HospitalAClient{BaseURL: cfg.HospitalABaseURL, HTTPClient: &http.Client{Timeout: 10 * time.Second}},
	}
	patientHandler := &handlers.PatientHandler{Service: patientService}

	routes.SetupRoutes(r, staffHandler, patientHandler, jwtUtil)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	log.Fatal(srv.ListenAndServe())
}

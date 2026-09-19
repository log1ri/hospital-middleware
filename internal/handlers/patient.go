package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/middlewares"
	"hospital-middleware/internal/models"
	"hospital-middleware/internal/services"
)

type PatientHandler struct {
	Service *services.PatientService
}

func (h *PatientHandler) Search(c *gin.Context) {
	// Get the hospital from the request context.
	hospital, ok := c.Get(middlewares.HospitalKey)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Ensure the hospital is a string and not empty.
	hospitalName, ok := hospital.(string)
	if !ok || hospitalName == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	// Parse the query parameters into a PatientFilters struct.
	var filters models.PatientFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid search filters"})
		return
	}

	// Trim whitespace from all filter fields to ensure clean input.
	filters.NationalID = strings.TrimSpace(filters.NationalID)
	filters.PassportID = strings.TrimSpace(filters.PassportID)
	filters.FirstName = strings.TrimSpace(filters.FirstName)
	filters.MiddleName = strings.TrimSpace(filters.MiddleName)
	filters.LastName = strings.TrimSpace(filters.LastName)
	filters.PhoneNumber = strings.TrimSpace(filters.PhoneNumber)
	filters.Email = strings.TrimSpace(filters.Email)

	// Call the service to search for patients based on the provided filters.
	patients, err := h.Service.Search(c.Request.Context(), hospitalName, filters)
	switch {
	case errors.Is(err, services.ErrPatientNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
	case errors.Is(err, services.ErrUnsupportedHospital):
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported hospital"})
	case errors.Is(err, services.ErrPatientStore):
		c.JSON(http.StatusInternalServerError, gin.H{"error": "patient storage failed"})
	case err != nil:
		c.JSON(http.StatusBadGateway, gin.H{"error": "Hospital HIS unavailable"})
	default:
		// Always answer a search with an array; a nil slice would marshal as null
		// and force every client to handle two shapes for "nothing found".
		if patients == nil {
			patients = []models.Patient{}
		}
		c.JSON(http.StatusOK, patients)
	}
}

package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"hospital-middleware/internal/services"
)

type StaffHandler struct {
	Service *services.StaffService
}

type createStaffRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	Hospital string `json:"hospital" binding:"required"`
}

type loginStaffRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Hospital string `json:"hospital" binding:"required"`
}

func (h *StaffHandler) CreateStaff(c *gin.Context) {
	// Parse the incoming JSON request
	var req createStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Call the service to create the staff member
	err := h.Service.CreateStaff(req.Username, req.Password, req.Hospital)

	switch {
	case errors.Is(err, services.ErrUsernameTaken):
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create staff"})
	default:
		c.JSON(http.StatusCreated, gin.H{"message": "staff created"})
	}

}

func (h *StaffHandler) LoginStaff(c *gin.Context) {
	// Parse the incoming JSON request
	var req loginStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Call the service to log in the staff member
	token, err := h.Service.LoginStaff(req.Username, req.Password, req.Hospital)
	switch {
	case errors.Is(err, services.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not log in"})
	default:
		c.JSON(http.StatusOK, gin.H{"token": token})
	}

}

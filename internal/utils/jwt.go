package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JwtUtil struct {
	SecretKey  []byte
	Expiration time.Duration
}

type StaffClaims struct {
	StaffID  uint   `json:"staff_id"`
	Hospital string `json:"hospital"`
	jwt.RegisteredClaims
}

func (j *JwtUtil) GenerateToken(staffID uint, hospital string) (string, error) {

	// Create a new token object, specifying signing method and the claims
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, StaffClaims{
		StaffID:  staffID,
		Hospital: hospital,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.Expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})

	return token.SignedString(j.SecretKey)
}

func (j *JwtUtil) ParseToken(tokenString string) (*StaffClaims, error) {
	claims := &StaffClaims{}

	// Parse the token with the claims
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return j.SecretKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())

	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.StaffID == 0 || claims.Hospital == "" {
		return nil, errors.New("invalid staff claims")
	}
	return claims, nil
}

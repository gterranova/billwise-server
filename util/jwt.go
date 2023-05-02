package util

import (
	"time"

	"github.com/dgrijalva/jwt-go"
)

func CreateToken(issuer string, expirationTime time.Time) (string, error) {
	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		Issuer:    issuer,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(Config.ApiSecret))
}

func ParseToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(Config.ApiSecret), nil
	})

	if err != nil || !token.Valid {
		return "", err
	}

	claims := token.Claims.(*jwt.StandardClaims) // Casting the token.Claims to the struct jwt.StandardClaims

	return claims.Issuer, nil
}

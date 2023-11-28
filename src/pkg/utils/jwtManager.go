package jwtManager

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt"
)

var jwtKey = []byte(os.Getenv("TOKEN_SECRET"))

type Claims struct {
	UserId   string
	username string
	jwt.StandardClaims
}

func GenerateToken(userId string, username string) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)
	claims := &Claims{
		UserId:   userId,
		username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)

	return tokenString, err
}

func VerificateToken(tokenString string) (*jwt.Token, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	return token, err
}

package middleware

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/helpleness/IMChatAdmin/model"
	"time"
)

var jwtKey = []byte("GwqPasswoed")

type Claims struct {
	UserID uint
	jwt.RegisteredClaims
}

func ReleaseToken(user model.User) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour * 7)
	claims := Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "Gwq",
			Subject:   "user token",
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
func ParseToken(tokenString string) (*jwt.Token, *Claims, error) {
	claims := &Claims{}
	var token *jwt.Token
	var err error
	token, err = jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	return token, claims, err
}

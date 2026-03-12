package jwt

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var JwtKey = []byte(os.Getenv("JWT_SECRET"))

type JWTClaim struct {
	Code         string `json:"code"`
	Email        string `json:"email"`
	RoleCode     string `json:"roleCode"`
	Collection   string `json:"collection"`
	TenantId     string `json:"tenantId"`
	IsSuperAdmin bool   `json:"isSuperAdmin"`
	jwt.RegisteredClaims
}

/*
Function For Generateing JWT Token where the claims
takes the name,id,email as input and gets stored in the claims
Storage in the  JWT token
*/
func GenerateJWT(code, email, roleCode, collectionName, tenantId string, isSuperAdmin bool) (string, error) {
	// expMinutesStr := os.Getenv("JWT_EXP_MINUTES")
	// expMinutes, err := strconv.Atoi(expMinutesStr)
	// if err != nil || expMinutes <= 0 {
	// 	expMinutes = 60
	// }
	// expHours := time.Duration(expMinutes) * time.Minute
	expDaysStr := os.Getenv("JWT_EXP_DAY")
	expDays, err := strconv.Atoi(expDaysStr)
	log.Println("ExpirationDay: ", expDays)
	if err != nil || expDays <= 0 {
		expDays = 7 // default to 1 day
	}

	expDuration := time.Duration(expDays) * 24 * time.Hour
	log.Println("exp:", expDuration)
	claims := &JWTClaim{
		Code:         code,
		Email:        email,
		RoleCode:     roleCode,
		Collection:   collectionName,
		TenantId:     tenantId,
		IsSuperAdmin: isSuperAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expDuration)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtKey)
}

/*
Function For Verifying that the token is Valid or Not
*/
func ValidateToken(tokenString string) (*JWTClaim, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaim{}, func(token *jwt.Token) (interface{}, error) {
		return JwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*JWTClaim)
	if !ok || !token.Valid {
		return nil, err
	}
	return claims, nil
}

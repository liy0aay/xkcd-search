package aaa

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const adminRole = "superuser"

type AAA struct {
	secretKey       string
	users           map[string]string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	log             *slog.Logger
}

func New(tokenTTL time.Duration, log *slog.Logger) (AAA, error) {
	const adminUser = "ADMIN_USER"
	const adminPass = "ADMIN_PASSWORD"
	const secretKeyEnv = "JWT_SECRET_KEY"

	user, ok := os.LookupEnv(adminUser)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin user from enviroment")
	}
	password, ok := os.LookupEnv(adminPass)
	if !ok {
		return AAA{}, fmt.Errorf("could not get admin password from enviroment")
	}
	secretKey, ok := os.LookupEnv(secretKeyEnv)
	if !ok {
		return AAA{}, fmt.Errorf("could not get JWT secret key from enviroment")
	}

	return AAA{
		secretKey:       secretKey,
		users:           map[string]string{user: password},
		accessTokenTTL:  tokenTTL,
		refreshTokenTTL: 30 * 24 * time.Hour,
		log:             log,
	}, nil
}

func (a AAA) Login(name, password string) (accessToken string, refreshToken string, err error) {
	if name == "" {
		return "", "", errors.New("empty user")
	}
	savedPass, ok := a.users[name]
	if !ok {
		return "", "", errors.New("unknown user")
	}
	if savedPass != password {
		return "", "", errors.New("wrong password")
	}

	accessClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  adminRole,
		"name": name,
		"type": "access",
		"exp":  jwt.NewNumericDate(time.Now().Add(a.accessTokenTTL)),
		"iat":  jwt.NewNumericDate(time.Now()),
	})
	accessTokenStr, err := accessClaims.SignedString([]byte(a.secretKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to create access token: %w", err)
	}

	refreshClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  adminRole,
		"name": name,
		"type": "refresh",
		"exp":  jwt.NewNumericDate(time.Now().Add(a.refreshTokenTTL)),
		"iat":  jwt.NewNumericDate(time.Now()),
	})
	refreshTokenStr, err := refreshClaims.SignedString([]byte(a.secretKey))
	if err != nil {
		return "", "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	return accessTokenStr, refreshTokenStr, nil
}

func (a AAA) RefreshAccessToken(refreshTokenString string) (string, error) {
	token, err := jwt.Parse(refreshTokenString, func(token *jwt.Token) (any, error) {
		return []byte(a.secretKey), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

	if err != nil {
		a.log.Error("cannot parse refresh token", "error", err)
		return "", fmt.Errorf("cannot parse token")
	}

	if !token.Valid {
		a.log.Error("refresh token is invalid")
		return "", errors.New("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		a.log.Error("invalid token claims")
		return "", errors.New("invalid token claims")
	}
	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "refresh" {
		a.log.Error("invalid token type")
		return "", errors.New("invalid token type")
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		a.log.Error("no subject", "error", err)
		return "", errors.New("incomplete token")
	}
	if subject != adminRole {
		a.log.Error("not admin", "subject", subject)
		return "", errors.New("not authorized")
	}

	name, ok := claims["name"].(string)
	if !ok {
		return "", errors.New("no name in token")
	}

	newAccessClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  adminRole,
		"name": name,
		"type": "access",
		"exp":  jwt.NewNumericDate(time.Now().Add(a.accessTokenTTL)),
		"iat":  jwt.NewNumericDate(time.Now()),
	})

	return newAccessClaims.SignedString([]byte(a.secretKey))
}

func (a AAA) Verify(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(a.secretKey), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		a.log.Error("cannot parse token", "error", err)
		return fmt.Errorf("cannot parse token")
	}
	if !token.Valid {
		a.log.Error("token is invalid")
		return errors.New("token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		a.log.Error("invalid token claims")
		return errors.New("invalid token claims")
	}

	tokenType, ok := claims["type"].(string)
	if !ok || tokenType != "access" {
		a.log.Error("invalid token type, expected access")
		return errors.New("invalid token type")
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		a.log.Error("no subject", "error", err)
		return errors.New("incomplete token")
	}
	if subject != adminRole {
		a.log.Error("not admin", "subject", subject)
		return errors.New("not authorized")
	}
	return nil
}

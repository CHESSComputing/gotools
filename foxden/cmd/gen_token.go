package cmd

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenParameters holds all input parameters to generate fake token
type TokenParameters struct {
	Alg               string
	Iss               string
	Sub               string
	Aud               string
	CustomClaims      map[string]any
	ExpirationMinutes int
	Secret            string
	PrivateKeyPath    string
}

func generateTestToken(p TokenParameters) {
	now := time.Now().Unix()
	// Base claims
	claims := jwt.MapClaims{
		"iat": now,
		"exp": now + int64(p.ExpirationMinutes*60),
	}

	if p.Iss != "" {
		claims["iss"] = p.Iss
	}
	if p.Sub != "" {
		claims["sub"] = p.Sub
	}
	if p.Aud != "" {
		audList := strings.Split(p.Aud, ",")
		if len(audList) == 1 {
			claims["aud"] = audList[0]
		} else {
			claims["aud"] = audList
		}
	}

	// Inject custom claims under `custom_claims`
	claims["custom_claims"] = p.CustomClaims

	// Sign and output token
	var token string
	var err error

	switch p.Alg {
	case "HS256":
		if p.Secret == "" {
			exitWithError("HS256 requires -secret")
		}
		token, err = generateHS256Token(claims, p.Secret)

	case "RS256":
		if p.PrivateKeyPath == "" {
			exitWithError("RS256 requires -privatekey")
		}
		token, err = generateRS256Token(claims, p.PrivateKeyPath)

	default:
		if p.Secret == "" {
			exitWithError("HS256 requires -secret")
		}
		token, err = generateHS256Token(claims, "secret-salt")
	}

	if err != nil {
		exitWithError("Token generation error: " + err.Error())
	}

	fmt.Println(token)
}

func generateHS256Token(claims jwt.MapClaims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func generateRS256Token(claims jwt.MapClaims, keyPath string) (string, error) {
	privKey, err := loadPrivateKey(keyPath)
	if err != nil {
		return "", err
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privKey)
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM file")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, err // original
		}
		rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
		return rsaKey, nil
	}
	return key, nil
}

func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}

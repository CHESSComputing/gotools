package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Standard JWT fields
	alg := flag.String("alg", "HS256", "Signing algorithm: HS256 or RS256")
	iss := flag.String("iss", "", "Issuer")
	sub := flag.String("sub", "", "Subject")
	aud := flag.String("aud", "", "Comma-separated audience list")
	expMins := flag.Int("exp", 60, "Expiration time in minutes")
	now := time.Now().Unix()

	// Custom claims
	claimsJSON := flag.String("claims", "", "Custom claims JSON (e.g. '{\"user\":\"test\",\"scope\":\"read\"}')")

	// Secrets/Keys
	secret := flag.String("secret", "", "Secret key for HS256")
	privateKeyPath := flag.String("privatekey", "", "Path to RSA private key file for RS256")

	flag.Parse()

	// Base claims
	claims := jwt.MapClaims{
		"iat": now,
		"exp": now + int64(*expMins*60),
	}

	if *iss != "" {
		claims["iss"] = *iss
	}
	if *sub != "" {
		claims["sub"] = *sub
	}
	if *aud != "" {
		audList := strings.Split(*aud, ",")
		if len(audList) == 1 {
			claims["aud"] = audList[0]
		} else {
			claims["aud"] = audList
		}
	}

	// Inject custom claims under `custom_claims`
	if *claimsJSON != "" {
		var custom map[string]any
		if err := json.Unmarshal([]byte(*claimsJSON), &custom); err != nil {
			exitWithError("Invalid JSON for custom claims: " + err.Error())
		}
		claims["custom_claims"] = custom
	}

	// Sign and output token
	var token string
	var err error

	switch *alg {
	case "HS256":
		if *secret == "" {
			exitWithError("HS256 requires -secret")
		}
		token, err = generateHS256Token(claims, *secret)

	case "RS256":
		if *privateKeyPath == "" {
			exitWithError("RS256 requires -privatekey")
		}
		token, err = generateRS256Token(claims, *privateKeyPath)

	default:
		exitWithError("Unsupported algorithm: " + *alg)
	}

	if err != nil {
		exitWithError("Token generation error: " + err.Error())
	}

	fmt.Println("Generated JWT:")
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


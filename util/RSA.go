package util

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log"
	"strings"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

// InitRSAKeys loads RSA keys from environment variables
// Expects PEM-encoded keys (with proper BEGIN/END markers and newlines)
// Environment variables:
// - RSA_PRIVATE_KEY: Private key in PEM format
// - RSA_PUBLIC_KEY: Public key in PEM format
func InitRSAKeys() error {
	privPEM := getEnv("RSA_PRIVATE_KEY", "")
	pubPEM := getEnv("RSA_PUBLIC_KEY", "")

	if privPEM == "" {
		return errors.New("RSA_PRIVATE_KEY environment variable not set")
	}
	if pubPEM == "" {
		return errors.New("RSA_PUBLIC_KEY environment variable not set")
	}

	// Clean up the PEM strings: handle both \n literals and actual newlines
	privPEM = strings.ReplaceAll(privPEM, "\\n", "\n")
	pubPEM = strings.ReplaceAll(pubPEM, "\\n", "\n")

	// Parse private key
	privBlock, _ := pem.Decode([]byte(privPEM))
	if privBlock == nil {
		return errors.New("failed to decode private key PEM from RSA_PRIVATE_KEY - ensure it's properly formatted with BEGIN/END markers")
	}

	priv, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return errors.New("failed to parse private key: " + err.Error())
	}

	// Parse public key
	pubBlock, _ := pem.Decode([]byte(pubPEM))
	if pubBlock == nil {
		return errors.New("failed to decode public key PEM from RSA_PUBLIC_KEY - ensure it's properly formatted with BEGIN/END markers")
	}

	pub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return errors.New("failed to parse public key: " + err.Error())
	}

	publicKey = pub.(*rsa.PublicKey)
	privateKey = priv

	log.Println("RSA keys loaded from environment variables successfully")
	return nil
}

// GetPrivateKey returns the loaded private key
func GetPrivateKey() *rsa.PrivateKey {
	return privateKey
}

// GetPublicKey returns the loaded public key
func GetPublicKey() *rsa.PublicKey {
	return publicKey
}

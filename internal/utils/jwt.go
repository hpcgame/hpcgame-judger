package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func ExtractUIDFromJWT(tokenString string) (string, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode payload: %v", err)
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("failed to unmarshal payload: %v", err)
	}

	sub, ok := claims["userId"].(string)
	if !ok {
		return "", fmt.Errorf("userId field not found or is not a string")
	}

	return sub, nil
}

func GenRandomStateString() string {
	str := ""
	for len(str) < 32 {
		buf := make([]byte, 32-len(str))
		_, err := rand.Read(buf)
		if err != nil {
			panic(err)
		}
		str += base64.RawURLEncoding.EncodeToString(buf)
	}
	return str
}

func GenerateRandomString(length int, prefix string) string {
	// Calculate the number of bytes needed to generate the string
	byteLength := (length * 6) / 8 // Base64 encoding uses 6 bits per character
	if (length*6)%8 != 0 {
		byteLength++
	}

	// Generate random bytes
	randomBytes := make([]byte, byteLength)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	// Encode the random bytes to a base64 string
	randomString := base64.URLEncoding.EncodeToString(randomBytes)

	// Trim the string to the desired length
	return prefix + randomString[:length]
}

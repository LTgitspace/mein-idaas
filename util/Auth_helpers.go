package util

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2 parameters for password hashing - with environment variable support
var (
	argon2Time      uint32
	argon2Memory    uint32
	argon2Threads   uint8
	argon2KeyLength uint32
	argon2SaltLen   int
)

// Argon2Params holds parameters extracted from a hash
type Argon2Params struct {
	Time      uint32
	Memory    uint32
	Threads   uint8
	KeyLength uint32
}

// InitArgon2Params initializes Argon2 parameters from environment variables or defaults
func InitArgon2Params() {
	// Time parameter (iterations)
	timeStr := os.Getenv("ARGON2_TIME")
	if timeStr == "" {
		argon2Time = 3
	} else {
		if val, err := strconv.ParseUint(timeStr, 10, 32); err == nil {
			argon2Time = uint32(val)
		} else {
			argon2Time = 3
		}
	}

	// Memory parameter in KB (default: 64MB = 65536 KB)
	memoryStr := os.Getenv("ARGON2_MEMORY")
	if memoryStr == "" {
		argon2Memory = 64 * 1024
	} else {
		if val, err := strconv.ParseUint(memoryStr, 10, 32); err == nil {
			argon2Memory = uint32(val)
		} else {
			argon2Memory = 64 * 1024
		}
	}

	// Parallelism parameter (threads)
	threadsStr := os.Getenv("ARGON2_THREADS")
	if threadsStr == "" {
		argon2Threads = 4
	} else {
		if val, err := strconv.ParseUint(threadsStr, 10, 8); err == nil {
			argon2Threads = uint8(val)
		} else {
			argon2Threads = 4
		}
	}

	// Key length (hash output size in bytes)
	keyLenStr := os.Getenv("ARGON2_KEY_LENGTH")
	if keyLenStr == "" {
		argon2KeyLength = 32
	} else {
		if val, err := strconv.ParseUint(keyLenStr, 10, 32); err == nil {
			argon2KeyLength = uint32(val)
		} else {
			argon2KeyLength = 32
		}
	}

	// Salt length in bytes
	saltLenStr := os.Getenv("ARGON2_SALT_LENGTH")
	if saltLenStr == "" {
		argon2SaltLen = 16
	} else {
		if val, err := strconv.Atoi(saltLenStr); err == nil {
			argon2SaltLen = val
		} else {
			argon2SaltLen = 16
		}
	}

	// Log the initialized parameters
	fmt.Printf("[ARGON2] Initialized with: time=%d, memory=%dKB, threads=%d, keylen=%d, saltlen=%d\n",
		argon2Time, argon2Memory, argon2Threads, argon2KeyLength, argon2SaltLen)
}

// HashPassword hashes a plaintext password using argon2 with current global parameters
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("empty password")
	}

	// Generate random salt
	salt, err := generateSalt(argon2SaltLen)
	if err != nil {
		return "", err
	}

	// Hash password with current global argon2 parameters
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLength)

	// Combine salt and hash for storage with current parameters
	return encodeArgon2Hash(salt, hash), nil
}

// ComparePassword compares argon2 hashed password with plaintext
// CRITICAL: Uses parameters extracted from the stored hash, NOT global variables
func ComparePassword(hashed, plain string) error {
	salt, hash, params, err := decodeArgon2Hash(hashed)
	if err != nil {
		return err
	}

	// Hash the provided password with the STORED parameters from the hash
	// This ensures users can always login even if global parameters change
	computedHash := argon2.IDKey([]byte(plain), salt, params.Time, params.Memory, params.Threads, params.KeyLength)

	// Compare hashes
	if !constantTimeCompare(hash, computedHash) {
		return errors.New("invalid password")
	}

	return nil
}

// HashToken returns a SHA256 hex of the token string for safe DB storage
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// generateSalt generates a random salt for argon2
func generateSalt(length int) ([]byte, error) {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

// encodeArgon2Hash encodes salt and hash into a string format with current global parameters
func encodeArgon2Hash(salt, hash []byte) string {
	// Format: $argon2id$v=19$m=<memory>,t=<time>,p=<threads>$<salt_hex>$<hash_hex>
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Time,
		argon2Threads,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash))
}

// decodeArgon2Hash decodes the argon2 hash format back to salt, hash, and parameters
// Returns the parameters that were used to create this hash
func decodeArgon2Hash(encoded string) ([]byte, []byte, *Argon2Params, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, nil, errors.New("invalid argon2 hash format")
	}

	// Parse the parameters section: m=65536,t=3,p=4
	paramStr := parts[3]
	params, err := parseArgon2ParamString(paramStr)
	if err != nil {
		return nil, nil, nil, err
	}

	// Decode salt
	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, errors.New("invalid salt encoding")
	}

	// Decode hash
	hash, err := hex.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, errors.New("invalid hash encoding")
	}

	return salt, hash, params, nil
}

// parseArgon2ParamString parses the parameter string from hash format
// Format: m=65536,t=3,p=4
func parseArgon2ParamString(paramStr string) (*Argon2Params, error) {
	params := &Argon2Params{
		KeyLength: 32, // Default keylen
	}

	pairs := strings.Split(paramStr, ",")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		switch key {
		case "m":
			if v, err := strconv.ParseUint(val, 10, 32); err == nil {
				params.Memory = uint32(v)
			}
		case "t":
			if v, err := strconv.ParseUint(val, 10, 32); err == nil {
				params.Time = uint32(v)
			}
		case "p":
			if v, err := strconv.ParseUint(val, 10, 8); err == nil {
				params.Threads = uint8(v)
			}
		}
	}

	// Validate that we got the required parameters
	if params.Memory == 0 || params.Time == 0 || params.Threads == 0 {
		return nil, errors.New("invalid argon2 parameters in hash")
	}

	return params, nil
}

// constantTimeCompare compares two byte slices in constant time
func constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	result := 0
	for i := range a {
		result |= int(a[i]) ^ int(b[i])
	}
	return result == 0
}

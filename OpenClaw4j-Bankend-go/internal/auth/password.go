package auth

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

var ErrInvalidEncodedPassword = errors.New("invalid encoded password")

func VerifyPassword(password string, encodedPassword string) bool {
	ok, err := VerifyArgon2ID(password, encodedPassword)
	return err == nil && ok
}

func VerifyArgon2ID(password string, encodedPassword string) (bool, error) {
	parts := strings.Split(encodedPassword, "$")
	if len(parts) < 6 || parts[1] != "argon2id" {
		return false, ErrInvalidEncodedPassword
	}

	version, err := parseArgon2Version(parts[2])
	if err != nil {
		return false, err
	}
	if version != argon2.Version {
		return false, fmt.Errorf("%w: unsupported version %d", ErrInvalidEncodedPassword, version)
	}

	memory, iterations, parallelism, err := parseArgon2Params(parts[3])
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("%w: decode salt: %v", ErrInvalidEncodedPassword, err)
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("%w: decode hash: %v", ErrInvalidEncodedPassword, err)
	}

	actual := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expected)))
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return false, nil
	}
	return true, nil
}

func parseArgon2Version(versionPart string) (uint32, error) {
	versionPart = strings.TrimSpace(versionPart)
	if !strings.HasPrefix(versionPart, "v=") {
		return 0, ErrInvalidEncodedPassword
	}
	version, err := strconv.ParseUint(strings.TrimPrefix(versionPart, "v="), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%w: parse version: %v", ErrInvalidEncodedPassword, err)
	}
	return uint32(version), nil
}

func parseArgon2Params(paramPart string) (uint32, uint32, uint8, error) {
	params := strings.Split(paramPart, ",")
	if len(params) != 3 {
		return 0, 0, 0, ErrInvalidEncodedPassword
	}

	memory, err := parseArgon2Int(params[0], "m=")
	if err != nil {
		return 0, 0, 0, err
	}
	iterations, err := parseArgon2Int(params[1], "t=")
	if err != nil {
		return 0, 0, 0, err
	}
	parallelism, err := parseArgon2Int(params[2], "p=")
	if err != nil {
		return 0, 0, 0, err
	}

	return memory, iterations, uint8(parallelism), nil
}

func parseArgon2Int(value string, prefix string) (uint32, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, prefix) {
		return 0, ErrInvalidEncodedPassword
	}
	parsed, err := strconv.ParseUint(strings.TrimPrefix(value, prefix), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%w: parse %s: %v", ErrInvalidEncodedPassword, prefix, err)
	}
	return uint32(parsed), nil
}

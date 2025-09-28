package utils

import (
	"math/rand"
	"strings"
)

func StringSliceContains(s []string, e string) bool {
	for _, a := range s {
		if strings.EqualFold(a, e) {
			return true
		}
	}
	return false
}

// StringSliceHasAll checks if all elements in slice 'e' are present in slice 's'.
//
// Parameters:
// - s: the slice to check against
// - e: the slice containing elements to check
// Return type: bool
func StringSliceHasAll(s []string, e []string) bool {
	for _, a := range e {
		if !StringSliceContains(s, a) {
			return false
		}
	}
	return true
}

// StringSliceContainsOneOf checks if any element in slice 'e' is present in slice 's'.
//
// Parameters:
// - s: the slice to check against
// - e: the slice containing elements to check
// Return type: bool
func StringSliceContainsOneOf(s []string, e []string) bool {
	for _, a := range e {
		if StringSliceContains(s, a) {
			return true
		}
	}
	return false
}

// MapContainsKey checks if the given map contains the specified key.
//
// Parameters:
// - m: the map to search in
// - k: the key to search for
// Return type: bool
// Returns true if the map contains the key, false otherwise.
func MapContainsKey(m map[string]string, k string) bool {
	for key := range m {
		if key == k {
			return true
		}
	}

	return false
}

// MapContainsValue checks if the given map contains the specified value.
//
// Parameters:
// - m: the map to search in
// - v: the value to search for
// Return type: bool
// Returns true if the map contains the value, false otherwise.
func MapContainsValue(m map[string]string, v string) bool {
	for _, value := range m {
		if value == v {
			return true
		}
	}

	return false
}

// GenerateRandomAlphaNumeric generates a random alphanumeric string of the specified size.
//
// Parameters:
// - size: the length of the generated alphanumeric string
// Return type: string
func GenerateRandomAlphaNumeric(size int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// ToPointer converts a value of type T to a pointer of type *T.
//
// Parameters:
// - v: the value to convert
func ToPointer[T any](v T) *T {
	return &v
}

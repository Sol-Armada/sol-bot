package utils

import "strings"

func StringSliceContains(s []string, e string) bool {
	for _, a := range s {
		if strings.ToUpper(a) == strings.ToUpper(e) {
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

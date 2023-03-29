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

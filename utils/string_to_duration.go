package utils

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

func StringToDuration(str string) (time.Duration, error) {
	// if the str only has a number, then it's in minutes
	rgx := regexp.MustCompile(`^\d+$`)
	if rgx.MatchString(str) {
		str += "m"
	}

	// convert 'd' to 'h'
	rgx = regexp.MustCompile(`(\d+)d`)
	if rgx.MatchString(str) {
		s := rgx.FindStringSubmatch(str)
		i, err := strconv.Atoi(s[1])
		if err != nil {
			return 0, err
		}
		str = strings.Replace(str, s[0], strconv.Itoa(i*24)+"h", 1)
	}

	duration, err := time.ParseDuration(str)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

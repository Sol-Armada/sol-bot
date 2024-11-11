package config

import (
	"errors"

	"github.com/sol-armada/sol-bot/stores"
)

func GetAttendanceTags() ([]string, error) {
	c, ok := stores.Get().GetConfigsStore()
	if !ok {
		return nil, errors.New("could not get configs store")
	}

	res := c.Get("tags")
	if res.Err() != nil {
		return nil, res.Err()
	}

	var out []string
	if err := res.Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

package config

import (
	"errors"
	"strings"

	"github.com/sol-armada/sol-bot/stores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetAttendanceTags() ([]string, error) {
	c, ok := stores.Get().GetConfigsStore()
	if !ok {
		return nil, errors.New("could not get configs store")
	}

	res := c.Get("attendance_tags")
	if res.Err() != nil {
		return nil, res.Err()
	}

	var out []string
	if err := res.Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

func NewAttendanceTag(tag string) error {
	c, ok := stores.Get().GetConfigsStore()
	if !ok {
		return errors.New("could not get configs store")
	}

	tags, err := GetAttendanceTags()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	for _, t := range tags {
		if t == tag {
			return nil
		}
	}

	tags = append(tags, strings.ToUpper(strings.ReplaceAll(tag, " ", "-")))
	return c.Upsert("attendance_tags", tags)
}

func GetAttendanceNames() ([]string, error) {
	raw, err := GetConfig("attendance_names")
	if err != nil {
		return nil, err
	}

	names, ok := raw.(bson.A)
	if !ok {
		return nil, errors.New("could not convert attendance names to []string")
	}

	var out []string
	for _, name := range names {
		out = append(out, name.(string))
	}

	return out, nil
}

func ValidAttendanceName(name string) (bool, error) {
	names, err := GetAttendanceNames()
	if err != nil {
		return false, err
	}

	name = strings.ToLower(name)
	for _, n := range names {
		if strings.ToLower(n) == name {
			return true, nil
		}
	}

	return false, nil
}

func NewAttendanceName(name string) error {
	c, ok := stores.Get().GetConfigsStore()
	if !ok {
		return errors.New("could not get configs store")
	}

	names, err := GetAttendanceNames()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	for _, n := range names {
		if n == name {
			return nil
		}
	}

	names = append(names, name)
	return c.Upsert("attendance_names", names)
}

func RemoveAttendanceName(name string) error {
	c, ok := stores.Get().GetConfigsStore()
	if !ok {
		return errors.New("could not get configs store")
	}

	names, err := GetAttendanceNames()
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	for i, n := range names {
		if n == name {
			names = append(names[:i], names[i+1:]...)
			break
		}
	}

	return c.Upsert("attendance_names", names)
}

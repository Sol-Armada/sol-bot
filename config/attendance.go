package config

// Stub: attendance config APIs now return sensible defaults or empty arrays.
// TODO: Move attendance names and tags to settings files or a dedicated package.

func GetAttendanceTags() ([]string, error) {
	return []string{}, nil
}

func NewAttendanceTag(tag string) error {
	_ = tag
	return nil
}

func GetAttendanceNames() ([]string, error) {
	return []string{}, nil
}

func ValidAttendanceName(name string) (bool, error) {
	_ = name
	return true, nil
}

func NewAttendanceName(name string) error {
	_ = name
	return nil
}

func RemoveAttendanceName(name string) error {
	_ = name
	return nil
}


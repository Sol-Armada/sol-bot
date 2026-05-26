package attendance

import "errors"

func ListActive(limit int) ([]*Attendance, error) {
	if attendanceStore == nil {
		return nil, errors.New("attendance store not found")
	}
	return attendanceStore.ListActive(limit)
}

func ListRecorded(limit int) ([]*Attendance, error) {
	if attendanceStore == nil {
		return nil, errors.New("attendance store not found")
	}
	return attendanceStore.ListRecorded(limit)
}

func List(filter any, limit int, page int) ([]*Attendance, error) {
	if attendanceStore == nil {
		return nil, errors.New("attendance store not found")
	}
	_ = filter
	return attendanceStore.List(limit, page)
}

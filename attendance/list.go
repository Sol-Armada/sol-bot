package attendance

import "errors"

func ListActive(limit int) ([]*Attendance, error) {
	if attendanceStore == nil {
		return nil, errors.New("attendance store not found")
	}
	return attendanceStore.ListActive(limit)
}

func List(limit int, page int) ([]*Attendance, error) {
	if attendanceStore == nil {
		return nil, errors.New("attendance store not found")
	}
	return attendanceStore.List(limit, page)
}

func ListByIds(ids []string) ([]*Attendance, error) {
	if attendanceStore == nil {
		return nil, errors.New("attendance store not found")
	}
	return attendanceStore.ListByIds(ids)
}

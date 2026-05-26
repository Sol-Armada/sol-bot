package attendance

import "errors"

func GetUniqueMemberCount(days int) (int, error) {
	if attendanceStore == nil {
		return 0, errors.New("attendance store not found")
	}
	return attendanceStore.GetUniqueMemberCount(days)
}

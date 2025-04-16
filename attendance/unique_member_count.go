package attendance

func GetUniqueMemberCount(days int) (int, error) {
	return attendanceStore.GetUniqueMemberCount(days)
}

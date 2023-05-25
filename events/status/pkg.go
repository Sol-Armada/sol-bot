package status

type Status int

const (
	Created Status = iota
	Announced
	Notified_DAY
	Notified_HOUR
	Live
	Finished
	Cancelled
	Deleted
)

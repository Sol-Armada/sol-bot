package bot

import "errors"

var (
	ChannelNotExist         error = errors.New("channel does not exist")
	InvalidPermissions      error = errors.New("invalid permissions")
	InvalidSubcommand       error = errors.New("invalid subcommand")
	InvalidAttendanceRecord error = errors.New("invalid attendance record")
)

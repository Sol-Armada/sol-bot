package dbnotify

const (
	ChannelMembers    = "solbot_members"
	ChannelAttendance = "solbot_attendance"
	ChannelTokens     = "solbot_tokens"
	ChannelEvents     = "solbot_events"
	ChannelDocs       = "solbot_docs"
	ChannelRSI        = "solbot_rsi"
	ChannelLogs       = "solbot_logs"
)

// AllChannels returns every built-in domain channel emitted by database triggers.
func AllChannels() []string {
	return []string{
		ChannelMembers,
		ChannelAttendance,
		ChannelTokens,
		ChannelEvents,
		ChannelDocs,
		ChannelRSI,
		ChannelLogs,
	}
}

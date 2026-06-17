package dbnotify

type Channel string

const (
	ChannelMembers    Channel = "solbot_members"
	ChannelAttendance Channel = "solbot_attendance"
	ChannelTokens     Channel = "solbot_tokens"
	ChannelEvents     Channel = "solbot_events"
	ChannelDocs       Channel = "solbot_docs"
	ChannelRSI        Channel = "solbot_rsi"
	ChannelLogs       Channel = "solbot_logs"
)

// AllChannels returns every built-in domain channel emitted by database triggers
func AllChannels() []Channel {
	return []Channel{
		ChannelMembers,
		ChannelAttendance,
		ChannelTokens,
		ChannelEvents,
		ChannelDocs,
		ChannelRSI,
		ChannelLogs,
	}
}

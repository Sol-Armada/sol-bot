package systemd

import (
	"log/slog"
	"net"
	"os"
)

// Notify sends a notification to systemd
func Notify(state string) error {
	socketPath := os.Getenv("NOTIFY_SOCKET")
	if socketPath == "" {
		// Not running under systemd, skip notification
		slog.Debug("NOTIFY_SOCKET not set, skipping systemd notification")
		return nil
	}

	// Connect to the systemd socket
	conn, err := net.Dial("unixgram", socketPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send the notification
	_, err = conn.Write([]byte(state))
	if err != nil {
		return err
	}

	slog.Debug("sent systemd notification", "state", state)
	return nil
}

// Ready notifies systemd that the service is ready
func Ready() error {
	return Notify("READY=1")
}

// Stopping notifies systemd that the service is stopping
func Stopping() error {
	return Notify("STOPPING=1")
}

// Status sends a status message to systemd
func Status(message string) error {
	return Notify("STATUS=" + message)
}

// Watchdog sends a watchdog ping to systemd
func Watchdog() error {
	return Notify("WATCHDOG=1")
}

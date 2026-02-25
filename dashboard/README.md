# Sol-Bot Dashboard

An internal web dashboard for monitoring and managing the Sol-Bot Discord bot.

## Features

The dashboard provides real-time monitoring and insights for:

### Overview (Home)
- Total members, active events, tokens in circulation
- Member breakdown by rank
- Recent attendance records
- Active raffles and giveaways
- 24-hour activity summaries

### Members Management
- Total, validated, and unvalidated member counts
- Pending onboarding queue
- Recent joins/leaves tracking
- Member search functionality
- RSI affiliation status
- Rank distribution

### Attendance & Events
- Active and historical event tracking
- Success rate metrics
- Average attendee counts
- Event details and status

### Token Economy
- Total tokens in circulation
- Top token holders leaderboard
- Recent token distributions
- Distribution breakdown by reason (attendance, raffles, etc.)

### Activity Monitoring
- Members active today/this week
- Voice channel activity (24h)
- Message activity (24h)
- Star Citizen gameplay tracking

### Raffles & Giveaways
- Active and recent raffles
- Upcoming/recent giveaways
- Entry counts and winners
- End time tracking

## Technology

- **Backend**: Go with WebSocket support
- **Frontend**: HTML templates with Pico CSS
- **Real-time Updates**: WebSocket connection for live metric updates every 5 seconds
- **Database**: MongoDB (same as main bot)

## Configuration

Add to your `settings.toml`:

```toml
[DASHBOARD]
port = 8080  # Port to run dashboard on (default: 8080)
```

## Access

The dashboard runs on `http://localhost:8080` (or configured port) and is accessible only on the local network. Since it's designed for internal use, no authentication is required.

### Routes

- `/` - Dashboard home/overview
- `/members` - Member management
- `/attendance` - Attendance tracking
- `/tokens` - Token economy
- `/activity` - Activity monitoring
- `/raffles` - Raffles and giveaways
- `/ws` - WebSocket endpoint for live updates
- `/api/metrics` - JSON metrics endpoint
- `/api/members/search?q=name` - Member search API

## Architecture

### Extensibility

The dashboard is designed to be easily extensible:

#### Adding New Metrics

1. Add metric fields to structs in `dashboard/metrics.go`
2. Implement collection logic in `collect*Metrics()` functions
3. Update templates to display new metrics
4. WebSocket automatically broadcasts updates

#### Adding New Pages

1. Create handler in `dashboard/handlers.go`
2. Register route in `registerRoutes()`
3. Create template in `dashboard/templates/`
4. Add navigation link to `base.html`

#### Adding New API Endpoints

1. Create handler function in `dashboard/handlers.go`
2. Register route in `registerRoutes()`
3. Return JSON with appropriate content type

### WebSocket Protocol

Messages sent to clients:

```json
{
  "type": "metrics_update",
  "data": {
    // Full Metrics object
  }
}
```

Clients can listen for the `metrics-update` DOM event to handle updates:

```javascript
window.addEventListener('metrics-update', function(event) {
  const metrics = event.detail;
  // Update UI with new metrics
});
```

## Development

### Local Testing

```bash
# Build
go build -o bin/solbot-test ./cmd/

# Run (requires MongoDB and Discord configuration)
./bin/solbot-test
```

### Adding Custom Styles

While Pico CSS provides the base styling, you can add custom CSS in the `<style>` section of templates or create a separate CSS file.

### Template Structure

- `base.html` - Base layout with navigation and WebSocket setup
- `home.html` - Dashboard overview
- `members.html` - Member management page
- `attendance.html` - Attendance tracking page
- `tokens.html` - Token economy page
- `activity.html` - Activity monitoring page
- `raffles.html` - Raffles and giveaways page

Each page template extends the base and defines:
- `content` block - Page content
- `scripts` block - Page-specific JavaScript

## Security Considerations

Since this dashboard is for internal use only:

- No authentication is implemented
- Should only be accessible on internal network
- Do not expose port publicly
- Consider using a reverse proxy with authentication if needed for remote access
- Use firewall rules to restrict access to trusted IPs

## Future Enhancements

Potential additions:
- Historical trend charts (Chart.js integration)
- Export data functionality
- Admin actions (approve onboarding, manage tokens, etc.)
- Alert notifications
- Custom date range filters
- Member profile detailed view
- Attendance event creation/management

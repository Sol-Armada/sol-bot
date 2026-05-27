#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <vmid> <phase>" >&2
  exit 1
fi

VMID="$1"
PHASE="$2"

# Run bootstrap only after the container is started.
if [[ "$PHASE" != "post-start" ]]; then
  exit 0
fi

# Defaults can be overridden by setting these vars in this script.
REPO_OWNER="${REPO_OWNER:-Sol-Armada}"
REPO_NAME="${REPO_NAME:-sol-bot}"
REPO_REF="${REPO_REF:-db-migration}"
RUN_UPDATER_NOW="${RUN_UPDATER_NOW:-true}"

BASE_URL="https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/${REPO_REF}"
UPDATER_SCRIPT_URL="${BASE_URL}/ops/updater/solbot-updater.sh"
UPDATER_ENV_URL="${BASE_URL}/ops/updater/updater.env.example"
SOLBOT_SERVICE_URL="${BASE_URL}/systemd/solbot.service"
UPDATER_SERVICE_URL="${BASE_URL}/systemd/solbot-updater.service"
UPDATER_TIMER_URL="${BASE_URL}/systemd/solbot-updater.timer"
SENTINEL_FILE="/var/lib/solbot/.updater-bootstrapped"

log() {
  echo "[solbot-hook:${VMID}] $*"
}

ct_exec() {
  pct exec "$VMID" -- bash -lc "$1"
}

if ! pct status "$VMID" | grep -q "status: running"; then
  log "container is not running"
  exit 1
fi

log "waiting for container network"
for i in {1..30}; do
  if ct_exec "getent hosts github.com >/dev/null 2>&1"; then
    break
  fi

  if [[ "$i" -eq 30 ]]; then
    log "network did not become ready in time"
    exit 1
  fi

  sleep 2
done

log "installing bootstrap dependencies"
ct_exec "export DEBIAN_FRONTEND=noninteractive; apt-get update -y && apt-get install -y curl jq ca-certificates"

log "ensuring solbot runtime user and paths"
ct_exec "getent group solbot >/dev/null || groupadd --system solbot"
ct_exec "id -u solbot >/dev/null 2>&1 || useradd --system --gid solbot --home-dir /var/lib/solbot --shell /usr/sbin/nologin solbot"
ct_exec "install -d -m 755 /var/lib/solbot /etc/solbot /opt/solbot-releases"
ct_exec "chown -R solbot:solbot /var/lib/solbot /opt/solbot-releases"

log "installing updater assets"
ct_exec "install -d -m 755 /usr/local/bin /etc/systemd/system /etc/solbot /var/lib/solbot /opt/solbot-releases"
ct_exec "curl -fsSL '${UPDATER_SCRIPT_URL}' -o /usr/local/bin/solbot-updater.sh"
ct_exec "chmod 755 /usr/local/bin/solbot-updater.sh"
ct_exec "curl -fsSL '${UPDATER_SERVICE_URL}' -o /etc/systemd/system/solbot-updater.service"
ct_exec "chmod 644 /etc/systemd/system/solbot-updater.service"
ct_exec "curl -fsSL '${UPDATER_TIMER_URL}' -o /etc/systemd/system/solbot-updater.timer"
ct_exec "chmod 644 /etc/systemd/system/solbot-updater.timer"

# Preserve any existing updater.env to avoid overwriting secrets/token.
ct_exec "if [[ ! -f /etc/solbot/updater.env ]]; then curl -fsSL '${UPDATER_ENV_URL}' -o /etc/solbot/updater.env; fi"
ct_exec "chmod 600 /etc/solbot/updater.env"
ct_exec "chown -R solbot:solbot /var/lib/solbot /opt/solbot-releases /etc/solbot"

log "installing solbot service"
ct_exec "curl -fsSL '${SOLBOT_SERVICE_URL}' -o /etc/systemd/system/solbot.service"
ct_exec "chmod 644 /etc/systemd/system/solbot.service"

log "enabling solbot and updater services"
ct_exec "systemctl daemon-reload"
ct_exec "systemctl enable solbot.service"
ct_exec "systemctl enable --now solbot-updater.timer"

if [[ "$RUN_UPDATER_NOW" == "true" ]]; then
  log "triggering immediate updater run"
  ct_exec "systemctl start solbot-updater.service"
else
  log "skipping immediate updater run (RUN_UPDATER_NOW=false)"
fi

ct_exec "date -Iseconds > '${SENTINEL_FILE}'"
log "bootstrap completed"

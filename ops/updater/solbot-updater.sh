#!/usr/bin/env bash
set -euo pipefail

REPO_OWNER="${REPO_OWNER:-Sol-Armada}"
REPO_NAME="${REPO_NAME:-sol-bot}"
ARCH="${ARCH:-amd64}"
APP_NAME="${APP_NAME:-solbot}"
SERVICE_NAME="${SERVICE_NAME:-solbot}"
INSTALL_PATH="${INSTALL_PATH:-/opt/solbot}"
RELEASES_DIR="${RELEASES_DIR:-/opt/solbot-releases}"
STATE_DIR="${STATE_DIR:-/var/lib/solbot}"

LOCK_FILE="/var/lock/${APP_NAME}-updater.lock"
TMP_DIR="$(mktemp -d)"
VERSION_FILE="${STATE_DIR}/current-version"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

require_cmd curl
require_cmd jq
require_cmd sha256sum
require_cmd systemctl
require_cmd flock

mkdir -p "${RELEASES_DIR}" "${STATE_DIR}" /var/lock

exec 9>"${LOCK_FILE}"
if ! flock -n 9; then
  echo "update already running"
  exit 0
fi

API_URL="https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest"
AUTH_HEADERS=()

release_json="$(curl -fsSL "${AUTH_HEADERS[@]}" -H "Accept: application/vnd.github+json" "${API_URL}")"
latest_tag="$(printf '%s' "${release_json}" | jq -r '.tag_name')"

if [[ -z "${latest_tag}" || "${latest_tag}" == "null" ]]; then
  echo "failed to determine latest release tag" >&2
  exit 1
fi

current_tag=""
if [[ -f "${VERSION_FILE}" ]]; then
  current_tag="$(cat "${VERSION_FILE}")"
fi

if [[ "${current_tag}" == "${latest_tag}" ]]; then
  echo "already up to date (${latest_tag})"
  exit 0
fi

asset_name="${APP_NAME}-linux-${ARCH}"
asset_url="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${latest_tag}/${asset_name}"
checksum_url="${asset_url}.sha256"

curl -fsSL "${AUTH_HEADERS[@]}" -o "${TMP_DIR}/${asset_name}" "${asset_url}"
curl -fsSL "${AUTH_HEADERS[@]}" -o "${TMP_DIR}/${asset_name}.sha256" "${checksum_url}"

(
  cd "${TMP_DIR}"
  sha256sum -c "${asset_name}.sha256"
)

release_dir="${RELEASES_DIR}/${latest_tag}"
install -d -m 755 "${release_dir}"
install -m 755 "${TMP_DIR}/${asset_name}" "${release_dir}/${APP_NAME}"

previous_release=""
if [[ -f "${VERSION_FILE}" ]]; then
  previous_release="$(cat "${VERSION_FILE}")"
fi

install -m 755 "${release_dir}/${APP_NAME}" "${INSTALL_PATH}"

if ! systemctl restart "${SERVICE_NAME}"; then
  echo "service restart failed, attempting rollback" >&2
  if [[ -n "${previous_release}" && -x "${RELEASES_DIR}/${previous_release}/${APP_NAME}" ]]; then
    install -m 755 "${RELEASES_DIR}/${previous_release}/${APP_NAME}" "${INSTALL_PATH}"
    systemctl restart "${SERVICE_NAME}" || true
  fi
  exit 1
fi

sleep 8
if ! systemctl is-active --quiet "${SERVICE_NAME}"; then
  echo "health check failed after update, attempting rollback" >&2
  if [[ -n "${previous_release}" && -x "${RELEASES_DIR}/${previous_release}/${APP_NAME}" ]]; then
    install -m 755 "${RELEASES_DIR}/${previous_release}/${APP_NAME}" "${INSTALL_PATH}"
    systemctl restart "${SERVICE_NAME}" || true
  fi
  exit 1
fi

echo "${latest_tag}" > "${VERSION_FILE}"

# Keep only the three newest release directories.
mapfile -t old_dirs < <(ls -1dt "${RELEASES_DIR}"/v* 2>/dev/null | tail -n +4 || true)
if [[ "${#old_dirs[@]}" -gt 0 ]]; then
  rm -rf "${old_dirs[@]}"
fi

echo "updated ${APP_NAME} to ${latest_tag}"

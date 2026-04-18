#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
timestamp=$(date +"%Y%m%d-%H%M%S")
results_dir=${TEST_RESULTS_DIR:-"test-results/hosted-auth/verify-hosted-auth-$timestamp"}
base_url=${BASE_URL-}
board_path=${BOARD_PATH:-/board}
settings_path=${SETTINGS_PATH:-/settings}
chrome_app=${CHROME_APP_NAME:-Google Chrome}
open_delay_seconds=${OPEN_DELAY_SECONDS:-6}
summary_path="$results_dir/summary.json"

if [ -z "$base_url" ]; then
  echo "BASE_URL is required for verify-hosted-auth-browser.sh" >&2
  exit 1
fi

if ! command -v osascript >/dev/null 2>&1; then
  echo "verify-hosted-auth-browser.sh requires osascript on macOS" >&2
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "verify-hosted-auth-browser.sh requires node on PATH" >&2
  exit 1
fi

mkdir -p "$results_dir"
base_url=${base_url%/}
board_url="$base_url$board_path"
settings_url="$base_url$settings_path"
head_sha=$(git -C "$repo_root" rev-parse HEAD 2>/dev/null || printf "")

escape_applescript_string() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

chrome_app_escaped=$(escape_applescript_string "$chrome_app")
board_url_escaped=$(escape_applescript_string "$board_url")
settings_url_escaped=$(escape_applescript_string "$settings_url")

osascript_output=$(
  osascript <<APPLESCRIPT
tell application "$chrome_app_escaped"
  activate
  set boardWindow to make new window
  set URL of active tab of boardWindow to "$board_url_escaped"
  delay $open_delay_seconds
  set boardActual to URL of active tab of boardWindow
  set boardTitle to title of active tab of boardWindow
  close boardWindow

  set settingsWindow to make new window
  set URL of active tab of settingsWindow to "$settings_url_escaped"
  delay $open_delay_seconds
  set settingsActual to URL of active tab of settingsWindow
  set settingsTitle to title of active tab of settingsWindow
  close settingsWindow
end tell

return boardActual & linefeed & boardTitle & linefeed & settingsActual & linefeed & settingsTitle
APPLESCRIPT
)

board_actual_url=$(printf '%s\n' "$osascript_output" | sed -n '1p')
board_title=$(printf '%s\n' "$osascript_output" | sed -n '2p')
settings_actual_url=$(printf '%s\n' "$osascript_output" | sed -n '3p')
settings_title=$(printf '%s\n' "$osascript_output" | sed -n '4p')

printf '%s\n' "$base_url" > "$results_dir/base-url.txt"
printf '%s\n' "$board_actual_url" > "$results_dir/board-url.txt"
printf '%s\n' "$board_title" > "$results_dir/board-title.txt"
printf '%s\n' "$settings_actual_url" > "$results_dir/settings-url.txt"
printf '%s\n' "$settings_title" > "$results_dir/settings-title.txt"

BASE_URL_VALUE=$base_url \
BOARD_PATH_VALUE=$board_path \
SETTINGS_PATH_VALUE=$settings_path \
CHROME_APP_VALUE=$chrome_app \
OPEN_DELAY_SECONDS_VALUE=$open_delay_seconds \
HEAD_SHA_VALUE=$head_sha \
BOARD_ACTUAL_URL_VALUE=$board_actual_url \
BOARD_TITLE_VALUE=$board_title \
SETTINGS_ACTUAL_URL_VALUE=$settings_actual_url \
SETTINGS_TITLE_VALUE=$settings_title \
SUMMARY_PATH_VALUE=$summary_path \
node --input-type=module -e '
  import fs from "node:fs";

  const normalizePath = (value) => value.replace(/\/+$/, "") || "/";
  const sameTarget = (actual, expected) => {
    try {
      const actualURL = new URL(actual);
      const expectedURL = new URL(expected);
      return (
        actualURL.origin === expectedURL.origin &&
        normalizePath(actualURL.pathname) === normalizePath(expectedURL.pathname)
      );
    } catch {
      return false;
    }
  };

  const expectedBoardURL = new URL(process.env.BOARD_PATH_VALUE, `${process.env.BASE_URL_VALUE}/`).toString();
  const expectedSettingsURL = new URL(process.env.SETTINGS_PATH_VALUE, `${process.env.BASE_URL_VALUE}/`).toString();
  const board = {
    expectedURL: expectedBoardURL,
    actualURL: process.env.BOARD_ACTUAL_URL_VALUE,
    title: process.env.BOARD_TITLE_VALUE,
    direct: sameTarget(process.env.BOARD_ACTUAL_URL_VALUE, expectedBoardURL),
  };
  const settings = {
    expectedURL: expectedSettingsURL,
    actualURL: process.env.SETTINGS_ACTUAL_URL_VALUE,
    title: process.env.SETTINGS_TITLE_VALUE,
    direct: sameTarget(process.env.SETTINGS_ACTUAL_URL_VALUE, expectedSettingsURL),
  };
  const summary = {
    status: board.direct && settings.direct ? "passed" : "failed",
    checkedAt: new Date().toISOString(),
    baseURL: process.env.BASE_URL_VALUE,
    headSha: process.env.HEAD_SHA_VALUE || null,
    chromeApp: process.env.CHROME_APP_VALUE,
    openDelaySeconds: Number(process.env.OPEN_DELAY_SECONDS_VALUE),
    board,
    settings,
  };

  fs.writeFileSync(process.env.SUMMARY_PATH_VALUE, `${JSON.stringify(summary, null, 2)}\n`);

  if (summary.status !== "passed") {
    process.stderr.write(
      `hosted browser auth verification failed: board=${board.actualURL} settings=${settings.actualURL}\n`
    );
    process.exit(1);
  }
' 

printf '%s\n' "$summary_path"

#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

version_file=${RELEASE_VERSION_FILE:-"$repo_root/VERSION"}
changelog_file=${RELEASE_CHANGELOG_FILE:-"$repo_root/CHANGELOG.md"}
release_tag=${RELEASE_TAG:-}
release_notes_output=${RELEASE_NOTES_OUTPUT:-}
semver_pattern='^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$'

fail() {
  printf '%s\n' "$1" >&2
  exit 1
}

[ -f "$version_file" ] || fail "missing VERSION file: $version_file"
[ -f "$changelog_file" ] || fail "missing CHANGELOG.md file: $changelog_file"

version=$(tr -d '\r' <"$version_file" | awk 'NF { print; exit }')
version=$(printf '%s' "$version" | awk '{$1=$1; print}')
[ -n "$version" ] || fail "VERSION file is empty: $version_file"

printf '%s' "$version" | grep -Eq "$semver_pattern" || fail "invalid semantic version in $version_file: $version"

expected_tag="v$version"
if [ -n "$release_tag" ] && [ "$release_tag" != "$expected_tag" ]; then
  fail "release tag $release_tag does not match VERSION $expected_tag"
fi

notes=$(
  awk -v version="$version" '
    BEGIN { in_section = 0 }
    $0 ~ "^## \\[" version "\\] - " {
      in_section = 1
      next
    }
    in_section && $0 ~ "^## \\[" {
      exit
    }
    in_section {
      sub(/\r$/, "")
      print
    }
  ' "$changelog_file"
)

notes_compact=$(printf '%s' "$notes" | tr -d '\r\n[:space:]')
[ -n "$notes_compact" ] || fail "CHANGELOG.md is missing a populated section for version $version"

if [ -n "$release_notes_output" ]; then
  notes_dir=$(dirname -- "$release_notes_output")
  mkdir -p "$notes_dir"
  printf '%s\n' "$notes" >"$release_notes_output"
fi

printf '%s\n' "$version"

#!/usr/bin/env bash

declare __DIRNAME="$(cd $(dirname "$0") && pwd)"

if ! command -v jq > /dev/null ; then
  echo "error: jq is required" >&2
  exit 1
fi

if ! command -v convert > /dev/null ; then
  echo "error: convert is required" >&2
  exit 1
fi

declare LIGHT_COLOR=gray20
declare DARK_COLOR=gray80

function main() {
  local -r font_file="$__DIRNAME/../fonts/NotoEmoji-VariableFont_wght.ttf"
  local -r emojis_path="$__DIRNAME"
  local -r dark_emojis_path="$emojis_path/dark"
  local -r light_emojis_path="$emojis_path/light"
  local -r emojis_file="$__DIRNAME/emojis.json"
  local code file
  while read -r emoji; do
    code="$(jq -r .emoji <<<"$emoji")"
    file="$(jq -r '"\(.code)_\(.file)"' <<<"$emoji")"
    # Light emoji
    convert -background none -fill "$LIGHT_COLOR" -font "$font_file" -pointsize 150 label:"$code" "$light_emojis_path/$file"
    # Dark emoji
    convert -background none -fill "$DARK_COLOR" -font "$font_file" -pointsize 150 label:"$code" "$dark_emojis_path/$file"
  done <<<"$(jq -c '.[]' "$emojis_file")"
  # Unknown emoji
  # Light emoji
  convert -background none -fill "$LIGHT_COLOR" -font "$font_file" -pointsize 150 label:"❔" "$light_emojis_path/unknown.png"
  # Dark emoji
  convert -background none -fill "$DARK_COLOR" -font "$font_file" -pointsize 150 label:"❔" "$dark_emojis_path/unknown.png"
}

main "$@"

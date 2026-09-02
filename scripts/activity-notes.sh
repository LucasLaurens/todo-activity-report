#!/bin/zsh
# Lance todo-activity-report pour la veille ouvrée et sauvegarde le rapport dans
# l'app Notes, sauf si une note pour cette date existe déjà.
# Planifié par ~/Library/LaunchAgents/activity-notes.plist
# (8h30 et 9h00 — la seconde exécution sert de rattrapage, la déduplication
# par titre de note empêche tout doublon).

set -euo pipefail

# launchd démarre avec un PATH minimal ; le CLI `claude` (synthèse IA) doit
# être trouvable.
export PATH="/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:$HOME/.local/bin"

# Rien en dur : le binaire, la personne et le dossier Notes se règlent par
# environnement, pour que le script soit utilisable tel quel.
BIN="${ACTIVITY_REPORT_BIN:-$HOME/projects/todo-activity-report/todo-activity-report}"
USER_FILTER="${ACTIVITY_REPORT_USER:?set ACTIVITY_REPORT_USER to the assignee to report on}"
NOTES_FOLDER="${ACTIVITY_REPORT_NOTES_FOLDER:-Résumés d'activité}"

# Veille ouvrée : sam/dim → rien ; lundi → vendredi ; sinon → hier.
weekday=$(date +%u) # 1=lundi … 7=dimanche
case "$weekday" in
6 | 7) exit 0 ;;
1) target=$(date -v-3d +%d/%m/%Y) ;;
*) target=$(date -v-1d +%d/%m/%Y) ;;
esac

title="Résumé d'activité — $target"

# Déjà sauvegardé ? (titre exact dans le dossier dédié)
exists=$(osascript - "$title" "$NOTES_FOLDER" <<'APPLESCRIPT'
on run argv
	set noteTitle to item 1 of argv
	set folderName to item 2 of argv
	tell application "Notes"
		if not (exists folder folderName) then return "no"
		if (count of (notes of folder folderName whose name is noteTitle)) > 0 then return "yes"
		return "no"
	end tell
end run
APPLESCRIPT
)
[[ "$exists" == "yes" ]] && exit 0

report=$("$BIN" --date="$target" --user="$USER_FILTER" --brief 2>/dev/null)
[[ -z "$report" ]] && exit 1

# Notes attend du HTML : échapper puis préserver la mise en forme en <pre>.
body=$(printf '%s' "$report" | sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g')

osascript - "$title" "$NOTES_FOLDER" "$body" <<'APPLESCRIPT'
on run argv
	set noteTitle to item 1 of argv
	set folderName to item 2 of argv
	set noteBody to item 3 of argv
	tell application "Notes"
		if not (exists folder folderName) then make new folder with properties {name:folderName}
		make new note at folder folderName with properties {name:noteTitle, body:"<h1>" & noteTitle & "</h1><pre>" & noteBody & "</pre>"}
	end tell
end run
APPLESCRIPT

#!/usr/bin/env bash
# Interaktives Terminal-Menu fuer die Wartung des Storyden-Servers.
# Reine Huelle um die Befehle aus WARTUNG-Ubuntu.md - fuer den Alltag per
# SSH (auch vom Handy aus), damit man sich die Befehle nicht merken muss.
#
#   cd ~/storyden && ./menu.sh
#
# Kein "set -e": ein fehlschlagender Befehl (z.B. docker compose) soll nur
# eine Fehlermeldung zeigen und ins Menu zurueckkehren, nicht das Skript
# beenden.
set -uo pipefail

cd "$(dirname "$0")"

COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_BOLD='\033[1m'
COLOR_RESET='\033[0m'

SERVICES=(storyden caddy postgres ddns)

pause() {
  echo
  read -rp "Weiter mit Enter... " _
}

confirm() {
  local word="$1"
  local reply
  read -rp "Zum Bestaetigen '$word' eingeben: " reply
  [[ "$reply" == "$word" ]]
}

header() {
  clear
  echo -e "${COLOR_BOLD}Storyden - Server-Verwaltung${COLOR_RESET}"
  echo "$(hostname)  |  $(date '+%Y-%m-%d %H:%M')"
  echo
  if command -v docker >/dev/null 2>&1; then
    local running total
    running="$(docker compose ps --status running -q 2>/dev/null | wc -l | tr -d ' ')"
    total="$(docker compose ps -q --all 2>/dev/null | wc -l | tr -d ' ')"
    if [[ "$total" -gt 0 && "$running" == "$total" ]]; then
      echo -e "Status: ${COLOR_GREEN}${running}/${total} Container laufen${COLOR_RESET}"
    elif [[ "$running" -gt 0 ]]; then
      echo -e "Status: ${COLOR_YELLOW}${running}/${total} Container laufen${COLOR_RESET}"
    else
      echo -e "Status: ${COLOR_RED}kein Container laeuft${COLOR_RESET}"
    fi
  fi
  echo
}

active_services() {
  docker compose ps --services 2>/dev/null
}

choose_service() {
  local prompt="$1"
  local -a active=()
  mapfile -t active < <(active_services)
  if [[ ${#active[@]} -eq 0 ]]; then
    active=("${SERVICES[@]}")
  fi

  echo "$prompt"
  local i=1
  for s in "${active[@]}"; do
    echo "  $i) $s"
    ((i++))
  done
  echo "  a) alle"
  echo "  0) Abbrechen"
  read -rp "> " choice

  if [[ "$choice" == "0" ]]; then
    return 1
  elif [[ "$choice" == "a" ]]; then
    echo "alle"
    return 0
  elif [[ "$choice" =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#active[@]} )); then
    echo "${active[$((choice - 1))]}"
    return 0
  else
    echo -e "${COLOR_RED}Ungueltige Auswahl.${COLOR_RESET}" >&2
    return 1
  fi
}

action_status() {
  docker compose ps
  echo
  docker system df
  echo
  df -h /
}

action_logs() {
  local svc
  svc="$(choose_service "Logs von welchem Dienst?")" || return
  echo "Strg+C beendet die Log-Ansicht und kehrt ins Menue zurueck."
  echo
  trap 'true' INT
  if [[ "$svc" == "alle" ]]; then
    docker compose logs -f --tail=100
  else
    docker compose logs -f --tail=100 "$svc"
  fi
  trap - INT
}

action_restart() {
  local svc
  svc="$(choose_service "Welchen Dienst neu starten?")" || return
  if [[ "$svc" == "alle" ]]; then
    docker compose restart
  else
    docker compose restart "$svc"
  fi
}

action_up() {
  echo "Wendet Aenderungen an .env / docker-compose.yml an (noetig statt"
  echo "'neu starten', wenn sich die Konfiguration geaendert hat)."
  echo
  docker compose up -d
}

action_switch_image() {
  echo "Verfuegbare Images:"
  docker images storyden-server --format '  {{.Tag}}  ({{.CreatedSince}})'
  echo
  read -rp "Tag eingeben (z.B. 2026-08-10) oder leer zum Abbrechen: " tag
  if [[ -z "$tag" ]]; then
    echo "Abgebrochen."
    return
  fi
  if ! docker image inspect "storyden-server:$tag" >/dev/null 2>&1; then
    echo -e "${COLOR_RED}Image storyden-server:$tag ist auf diesem Server nicht vorhanden.${COLOR_RESET}"
    return
  fi
  if [[ ! -f .env ]]; then
    echo -e "${COLOR_RED}.env nicht gefunden.${COLOR_RESET}"
    return
  fi
  sed -i.bak "s/^STORYDEN_IMAGE=.*/STORYDEN_IMAGE=storyden-server:${tag}/" .env
  echo "STORYDEN_IMAGE auf storyden-server:${tag} gesetzt."
  docker compose up -d storyden
}

action_backup_now() {
  bash backup.sh
}

action_list_backups() {
  if [[ -d backups ]]; then
    ls -lh backups/
  else
    echo "Noch keine Backups vorhanden (Verzeichnis backups/ fehlt)."
  fi
}

action_restore_backup() {
  if [[ ! -d backups ]]; then
    echo "Kein backups/-Verzeichnis vorhanden."
    return
  fi
  local -a dumps=()
  mapfile -t dumps < <(find backups -maxdepth 1 -name 'db-*.sql.gz' | sort -r)
  if [[ ${#dumps[@]} -eq 0 ]]; then
    echo "Keine Datenbank-Backups gefunden."
    return
  fi

  echo -e "${COLOR_RED}${COLOR_BOLD}ACHTUNG: Ueberschreibt die aktuelle Datenbank komplett.${COLOR_RESET}"
  echo "Verfuegbare Backups:"
  local i=1
  for d in "${dumps[@]}"; do
    echo "  $i) $d"
    ((i++))
  done
  echo "  0) Abbrechen"
  read -rp "> " choice
  if [[ "$choice" == "0" || -z "$choice" ]]; then
    echo "Abgebrochen."
    return
  fi
  if ! [[ "$choice" =~ ^[0-9]+$ ]] || (( choice < 1 || choice > ${#dumps[@]} )); then
    echo -e "${COLOR_RED}Ungueltige Auswahl.${COLOR_RESET}"
    return
  fi
  local dump="${dumps[$((choice - 1))]}"

  if ! confirm "WIEDERHERSTELLEN"; then
    echo "Abgebrochen."
    return
  fi

  gunzip -c "$dump" | docker compose exec -T postgres psql -U storyden storyden
  echo -e "${COLOR_GREEN}Datenbank aus $dump wiederhergestellt.${COLOR_RESET}"

  local assets="${dump/db-/assets-}"
  assets="${assets%.sql.gz}.tar.gz"
  if [[ -f "$assets" ]]; then
    read -rp "Zugehoerige Uploads aus $assets auch wiederherstellen? [j/N] " yn
    if [[ "$yn" =~ ^[jJ]$ ]]; then
      tar xzf "$assets" -C data
      sudo chown -R 1001:1001 data
      echo -e "${COLOR_GREEN}Uploads wiederhergestellt.${COLOR_RESET}"
    fi
  fi
}

action_psql() {
  echo "Postgres-Shell (\\q zum Beenden)."
  docker compose exec postgres psql -U storyden storyden
}

action_disk() {
  df -h /
  echo
  docker system df
  echo
  read -rp "Ungenutzte Docker-Image-Layer aufraeumen (docker image prune)? [j/N] " yn
  if [[ "$yn" =~ ^[jJ]$ ]]; then
    docker image prune -f
  fi
}

action_apt_upgrade() {
  echo "Fuehrt 'apt update && apt full-upgrade -y' aus."
  if ! confirm "UPGRADE"; then
    echo "Abgebrochen."
    return
  fi
  sudo apt update && sudo apt full-upgrade -y
  echo
  echo "Falls ein neuer Kernel installiert wurde: Menuepunkt 12 (Neustart) ausfuehren."
}

action_reboot() {
  echo -e "${COLOR_RED}${COLOR_BOLD}Startet den ganzen Server neu.${COLOR_RESET}"
  echo "Alle Container kommen dank 'restart: unless-stopped' von selbst wieder hoch."
  if ! confirm "REBOOT"; then
    echo "Abgebrochen."
    return
  fi
  sudo reboot
}

action_healthcheck() {
  if docker compose exec storyden wget -qO- http://127.0.0.1:8000/healthz; then
    echo
    echo -e "${COLOR_GREEN}Healthcheck OK.${COLOR_RESET}"
  else
    echo -e "${COLOR_RED}Healthcheck fehlgeschlagen.${COLOR_RESET}"
  fi
}

main_menu() {
  header
  cat <<'EOF'
 1) Status (Container, Speicherplatz)
 2) Logs ansehen
 3) Dienst neu starten
 4) Konfiguration anwenden (nach .env-Aenderung)
 5) Auf andere Image-Version wechseln / Rollback
 6) Backup jetzt erstellen
 7) Backups auflisten
 8) Backup wiederherstellen
 9) Postgres-Shell oeffnen
10) Speicherplatz / Aufraeumen
11) System-Update (apt)
12) Server neu starten
13) Health-Check
 0) Beenden
EOF
  echo
}

while true; do
  main_menu
  read -rp "> " choice
  echo
  case "$choice" in
    1) action_status ;;
    2) action_logs ;;
    3) action_restart ;;
    4) action_up ;;
    5) action_switch_image ;;
    6) action_backup_now ;;
    7) action_list_backups ;;
    8) action_restore_backup ;;
    9) action_psql ;;
    10) action_disk ;;
    11) action_apt_upgrade ;;
    12) action_reboot ;;
    13) action_healthcheck ;;
    0) echo "Tschuess."; exit 0 ;;
    *) echo -e "${COLOR_RED}Ungueltige Auswahl.${COLOR_RESET}" ;;
  esac
  pause
done

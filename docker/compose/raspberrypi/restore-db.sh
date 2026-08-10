#!/usr/bin/env bash
# Spielt den Datenbank-Dump vom Windows-PC in den Postgres-Container auf dem Pi.
#
#   bash restore-db.sh db.sql
#
# Laeuft nur in eine leere Datenbank. Ist schon etwas drin, muss --force
# angegeben werden - der Dump legt die Tabellen sonst ein zweites Mal an und
# bricht mittendrin ab.
set -euo pipefail

cd "$(dirname "$0")"

DUMP="${1:-db.sql}"
FORCE="${2:-}"

if [ ! -f "$DUMP" ]; then
  echo "Dump '$DUMP' nicht gefunden." >&2
  exit 1
fi

if [ ! -f .env ]; then
  echo ".env fehlt - erst Konfiguration anlegen." >&2
  exit 1
fi

env_value() {
  local key="$1" fallback="$2" value
  value="$(grep -E "^${key}=" .env | tail -n1 | cut -d= -f2- || true)"
  if [ -z "$value" ]; then echo "$fallback"; else echo "$value"; fi
}

PG_USER="$(env_value POSTGRES_USER storyden)"
PG_DB="$(env_value POSTGRES_DB storyden)"

echo "Starte Postgres ..."
docker compose up -d postgres

echo -n "Warte auf Datenbank"
for _ in $(seq 1 60); do
  if docker compose exec -T postgres pg_isready -U "$PG_USER" -d "$PG_DB" >/dev/null 2>&1; then
    echo " - bereit."
    break
  fi
  echo -n "."
  sleep 2
done

if ! docker compose exec -T postgres pg_isready -U "$PG_USER" -d "$PG_DB" >/dev/null 2>&1; then
  echo "" >&2
  echo "Datenbank ist nicht hochgekommen. Logs: docker compose logs postgres" >&2
  exit 1
fi

TABLES="$(docker compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -tAc \
  "select count(*) from information_schema.tables where table_schema = 'public'" | tr -d '[:space:]')"

if [ "$TABLES" != "0" ] && [ "$FORCE" != "--force" ]; then
  echo "" >&2
  echo "In '$PG_DB' liegen bereits $TABLES Tabellen." >&2
  echo "Entweder war der Restore schon erfolgreich, oder die App ist bereits" >&2
  echo "gestartet und hat das Schema selbst angelegt." >&2
  echo "" >&2
  echo "Sauber neu aufsetzen (loescht die Datenbank auf dem Pi):" >&2
  echo "  docker compose down" >&2
  echo "  docker volume rm storyden_postgres-data" >&2
  echo "  bash restore-db.sh $DUMP" >&2
  echo "" >&2
  echo "Trotzdem einspielen: bash restore-db.sh $DUMP --force" >&2
  exit 1
fi

echo "Spiele '$DUMP' ein ..."
docker compose exec -T postgres \
  psql -v ON_ERROR_STOP=1 -U "$PG_USER" -d "$PG_DB" < "$DUMP"

docker compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -c "ANALYZE" >/dev/null

echo ""
echo "Uebernommene Datensaetze (grob, groesste Tabellen):"
docker compose exec -T postgres psql -U "$PG_USER" -d "$PG_DB" -c \
  "select relname as tabelle, n_live_tup as zeilen
     from pg_stat_user_tables
    where n_live_tup > 0
    order by n_live_tup desc
    limit 10"

echo ""
echo "Restore fertig. Weiter mit: docker compose up -d"

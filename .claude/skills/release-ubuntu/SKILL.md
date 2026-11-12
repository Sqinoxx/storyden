---
name: release-ubuntu
description: Release the current Storyden (unidentists.de) version to the production Ubuntu server - build the amd64 image on this Windows PC, transfer only the image, back up the server, restart, and verify including a logged-in admin performance check. Use when the user asks to release, deploy, ship or update the live forum.
---

# Release Storyden to the Ubuntu server

Production: `unidentists.de`, server `192.168.178.105` (x86_64), SSH user `unidentist`,
compose project `~/storyden` (containers `storyden-storyden-1`, `storyden-postgres-1`,
`storyden-caddy-1`). Scripts live in `docker/compose/ubuntuserver/`. Human-facing
version of this process: `docker/compose/ubuntuserver/UPDATE-ohne-DB.md`.

Talk to the user in German. Order is fixed: **build → transfer image → backup → release → verify**.
Never restart the app before a verified backup exists.

## 0. Ask up front (one message)

The repo is PUBLIC. Never write passwords into files, commits or memory.

Ask for:
- the server's sudo password (needed for the backup, `backups/` is owned by root)
- admin login (username + password) for the post-release performance check

SSH itself works with the key, no password. Test with
`ssh -o BatchMode=yes unidentist@192.168.178.105 "echo ok"`.

## 1. Preflight

```powershell
cd F:\Zahnmedizin\Forum\storyden
git fetch origin main; git status -sb
```

- Uncommitted changes would ship (build marks commit as `-dirty`). Ask whether to commit
  them first. The build tag is `YYYY-MM-DD-b<commit count>`; rebuilding the same commit
  overwrites the tag on the server, which removes that rollback point. Commit first.
- Find what is new since the last release: `ssh ... "tail -3 ~/storyden/release-history.log"`
  (written in step 5; if missing, use `docker images storyden-server` dates) and
  `git log --oneline <last-commit>..HEAD -- web/src`.
- **Review new frontend code for the 2026-11-11 regression pattern**: any `<Menu.Root>`,
  `<Popover.Root>`, `<Dialog.Root>` etc. with `<Portal>` that is rendered once per list
  item (sidebar categories, thread cards, members) MUST have `lazyMount`, otherwise every
  hidden menu is rendered up front. Nested submenus listing the whole category tree inside
  each category menu produced 64,000 DOM nodes and made the site unusable for admins only.
  Use `lazyMount` but NOT `unmountOnExit` when the menu contains dialog/modal triggers
  (edit/delete/create items), or those dialogs close with the menu.
  Quick check: `git diff <last-commit>..HEAD -- web/src | Select-String "Menu.Root|Portal"`.

## 2. Build (PC)

Docker Desktop has only ~6 GB RAM. Building Go and Next.js in parallel crashed the daemon
(`failed to receive status: rpc error: code = Unavailable desc = error reading from server: EOF`).
Prebuild the Go stage first, then the full image. Run it with `run_in_background: true`
and **no output redirection**: in Windows PowerShell 5.1, `*>&1`/`2>&1` turns docker's
progress output on stderr into a terminating NativeCommandError under
`$ErrorActionPreference = "Stop"`.

```powershell
cd F:\Zahnmedizin\Forum\storyden
docker buildx build --platform linux/amd64 --provenance=false --sbom=false --target api-builder -f docker\all\Dockerfile .
if ($LASTEXITCODE -ne 0) { throw "api-builder failed" }
cd docker\compose\ubuntuserver; .\build-image.ps1
```

If the daemon still crashes: check `docker info`, then just retry. Success ends with
`Fertig: ...\transfer\storyden-amd64.tar (~190 MB)`.

## 3. Transfer the image only

```powershell
cd F:\Zahnmedizin\Forum\storyden\docker\compose\ubuntuserver
.\copy-to-server.ps1 -ServerUser unidentist -ImageOnly
```

**Never without `-ImageOnly`**: otherwise the PC's `.env` and old dumps overwrite the
server's. `docker load` does not touch the running container.

## 4. Backup (server, needs sudo)

PowerShell pipes stdin to native programs with a UTF-8 BOM and CRLF, so the password
arrives as `\xEF\xBB\xBFpassword\r\n` and sudo rejects it. Strip both on the server side.
`</dev/null` must be inside the `bash -c`, otherwise sudo itself gets no stdin.
Do not retry blindly after a failure: inspect with `"x" | ssh ... "od -c"` first.

```powershell
"<SUDO_PASSWORD>" | ssh -o BatchMode=yes unidentist@192.168.178.105 "tail -c +4 | tr -d '\r' | sudo -S -p '' bash -c 'bash /home/unidentist/storyden/backup.sh </dev/null'"
```

Takes ~2 minutes (assets are ~3 GB; the site can be sluggish meanwhile because of disk I/O).
Then verify with the stamp it printed (`db-YYYY-MM-DD-HHMM`):

```powershell
ssh -o BatchMode=yes unidentist@192.168.178.105 "cd ~/storyden/backups && ls -lh *<STAMP>* && gzip -t db-<STAMP>.sql.gz && gzip -t assets-<STAMP>.tar.gz && echo GZIP_OK && zcat db-<STAMP>.sql.gz | tail -3"
```

Required: both files present, DB ~3-4 MB (not tiny), assets ~3 GB, `GZIP_OK`, and the dump
ends with a `\unrestrict ...` line (pg_dump finished). Otherwise stop and report.

## 5. Release

```powershell
ssh -o BatchMode=yes unidentist@192.168.178.105 "cd ~/storyden && docker compose up -d 2>&1 && sleep 40 && docker compose ps --format '{{.Name}} {{.Image}} {{.Status}}' && docker inspect storyden-storyden-1 --format '{{.Image}}'"
```

- `storyden-storyden-1` must be `Up ... (healthy)`; its image ID must equal
  `docker images storyden-server:latest --format "{{.ID}}"` on the PC.
- `~/storyden/.env` must not pin `STORYDEN_IMAGE` to an old tag (check with grep).
- Record the release:
  `ssh ... "echo \"$(date -Is) <tag> <commit>\" >> ~/storyden/release-history.log"`

## 6. Verify

Use the Bash tool (Git Bash) for anything with nested quotes/JSON; PowerShell mangles them.

```bash
ssh -o BatchMode=yes unidentist@192.168.178.105 'cd ~/storyden; docker compose logs --since 5m storyden 2>&1 | grep -E "\"level\":\"(ERROR|WARN)\"" | cut -c1-250 | tail; curl -s -o /dev/null -k --resolve unidentists.de:443:127.0.0.1 -w "HTTP %{http_code} %{time_total}s\n" https://unidentists.de/'
```

Server-side request durations are in the app log (`"duration"` field). Normal: almost all
under 100 ms; one slow first request after restart is fine.

**Logged-in admin check (mandatory).** Anonymous checks missed the 2026-11-11 regression
because it only affected admins:

```powershell
$env:STORYDEN_USER="<admin>"; $env:STORYDEN_PASS="<password>"
node F:\Zahnmedizin\Forum\storyden\.claude\skills\release-ubuntu\perf-check.mjs F:\Zahnmedizin\Forum\storyden\web
Remove-Item Env:STORYDEN_USER, Env:STORYDEN_PASS
```

Healthy baseline (2026-11-12): DOM ~3,000 on `/`, ~7,000 on `/d`; longtasks max < 400 ms;
heap 35-60 MB; body portals 10-60. The script prints `RESULT: SLOW` (exit 1) above
DOM 10k / longtask 1 s / 300 portals / 150 MB. `pre-rendered submenus` shows which menu
renders eagerly - that is where to add `lazyMount`. If slow: fix, rebuild, transfer,
`docker compose up -d storyden` (no new backup needed for frontend-only fixes), re-check.

## Rollback

1. `docker images storyden-server` on the server, pick the previous tag, set
   `STORYDEN_IMAGE=storyden-server:<tag>` in `~/storyden/.env`.
2. The new version may have migrated the schema, so restore the step-4 backup:

```bash
cd ~/storyden
docker compose stop storyden
docker compose exec -T postgres psql -U storyden -d postgres -c "DROP DATABASE storyden WITH (FORCE);"
docker compose exec -T postgres psql -U storyden -d postgres -c "CREATE DATABASE storyden OWNER storyden;"
gunzip -c backups/db-<STAMP>.sql.gz | docker compose exec -T postgres psql -U storyden storyden
docker compose up -d storyden
```

Everything written to the forum after the release is lost. Confirm with the user first.

## Other pitfalls

- Never run `pnpm dev` or `go run ./cmd/backend` (CLAUDE.md). `task test:e2e:ci -- <spec>` is allowed.
- On Windows the e2e harness leaves `next start --port 3001` running, so the next run fails with
  `EADDRINUSE :::3001`. Before rerunning, kill only that process (check its command line
  contains `next` `start` `3001`).
- `tests/category/category-move-into.spec.ts` fails at `row.hover()` even on unchanged code
  (onboarding covers the collapsed sidebar on a fresh test DB). Not a release blocker.
- Foreground `Start-Sleep` is blocked; long steps go in `run_in_background` and you get notified.
- Never write DB dumps, `.env` copies or keys under tracked paths (`transfer/` is gitignored).
- Final report to the user: build tag + commit, backup file names/sizes, container status,
  perf-check numbers, anything skipped. Remind them to change any password they pasted into chat.

# Ermittelt die automatische Build-Nummer fuer die Admin-Anzeige.
# Nummer = Anzahl Git-Commits (steigt mit jedem Commit von selbst).

function Get-BuildInfo {
    param([Parameter(Mandatory)][string]$RepoRoot)

    $count = (git -C $RepoRoot rev-list --count HEAD).Trim()
    if ($LASTEXITCODE -ne 0) { throw "git rev-list fehlgeschlagen" }

    $commit = (git -C $RepoRoot rev-parse --short HEAD).Trim()
    if ($LASTEXITCODE -ne 0) { throw "git rev-parse fehlgeschlagen" }

    $dirty = (git -C $RepoRoot status --porcelain --untracked-files=no)
    if ($dirty) { $commit = "$commit-dirty" }

    [pscustomobject]@{
        Number = $count
        Commit = $commit
        Date   = (Get-Date -Format "yyyy-MM-dd")
    }
}

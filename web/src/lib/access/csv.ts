import type { AccessGrant, AccessGrantStatus } from "@/api/openapi-schema";

export type GrantCsvLabels = {
  name: string;
  handle: string;
  pass: string;
  reference: string;
  status: string;
  requested: string;
  decided: string;
  expires: string;
  decidedBy: string;
  note: string;
  reason: string;
};

// Spreadsheet apps run cells starting with these as formulas, and member names
// are free text.
const FORMULA_START = /^[=+\-@\t\r]/;

export function csvCell(value: string): string {
  const safe = FORMULA_START.test(value) ? `'${value}` : value;
  if (/[";\n\r]/.test(safe)) {
    return `"${safe.replace(/"/g, '""')}"`;
  }
  return safe;
}

function isoDate(iso: string | undefined): string {
  return iso ? iso.slice(0, 10) : "";
}

// Semicolon separated with a byte order mark, which is what German Excel
// opens correctly with a double click.
export function grantsToCsv(
  grants: AccessGrant[],
  labels: GrantCsvLabels,
  statusLabel: (s: AccessGrantStatus) => string,
): string {
  const header = [
    labels.name,
    labels.handle,
    labels.pass,
    labels.reference,
    labels.status,
    labels.requested,
    labels.decided,
    labels.expires,
    labels.decidedBy,
    labels.note,
    labels.reason,
  ];

  const rows = grants.map((g) => [
    g.account.name,
    g.account.handle,
    g.pass.name,
    g.reference_code,
    statusLabel(g.status),
    isoDate(g.requested_at),
    isoDate(g.decided_at),
    isoDate(g.expires_at),
    g.decided_by?.name ?? "",
    g.note,
    g.reason ?? "",
  ]);

  return (
    "﻿" +
    [header, ...rows].map((r) => r.map(csvCell).join(";")).join("\r\n") +
    "\r\n"
  );
}

export function downloadCsv(filename: string, content: string) {
  const blob = new Blob([content], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

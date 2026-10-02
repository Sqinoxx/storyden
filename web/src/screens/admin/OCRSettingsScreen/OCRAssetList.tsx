"use client";

import { useEffect, useState } from "react";

import { useAdminOCRAssetList } from "@/api/openapi-client/admin";
import { AdminOCRAssetList200AssetsItem } from "@/api/openapi-schema";
import { AdminOCRAssetListStatus } from "@/api/openapi-schema";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import * as Table from "@/components/ui/table";
import { useTranslation } from "@/lib/i18n";
import { Box, HStack, styled } from "@/styled-system/jsx";

type Filter = "open" | AdminOCRAssetListStatus;

const REFRESH_MS = 5000;

const reasonLabels: Record<string, string> = {
  "asset file is empty": "Datei ist leer (0 Byte)",
  "asset file not found on disk": "Datei fehlt auf dem Server",
  "ocr engine unavailable": "OCR-Engine nicht verfügbar",
  "unsupported mime type": "Dateityp nicht unterstützt",
  "pdf has no usable text layer and no rasteriser available":
    "PDF ohne Textebene, kein Rasterizer verfügbar",
};

function describeReason(error?: string): string {
  if (!error) return "–";
  if (reasonLabels[error]) return reasonLabels[error];

  const size = error.match(/^file exceeds max size of (\d+) MB$/);
  if (size) return `Größer als ${size[1]} MB`;

  const timeout = error.match(/^text extraction exceeded the (.+) timeout$/);
  if (timeout) return `Zeitlimit überschritten (${timeout[1]})`;

  const mime = error.match(/^unsupported mime type: (.+)$/);
  if (mime) return `Dateityp nicht unterstützt (${mime[1]})`;

  if (error.includes("truncated file"))
    return "Datei beschädigt oder unvollständig";

  return error;
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function formatElapsed(from: string, now: number): string {
  const seconds = Math.max(
    0,
    Math.floor((now - new Date(from).getTime()) / 1000),
  );
  if (seconds < 60) return `seit ${seconds} s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `seit ${minutes} min`;
  return `seit ${Math.floor(minutes / 60)} h ${minutes % 60} min`;
}

function formatTime(at: string): string {
  return new Date(at).toLocaleString("de-DE", {
    dateStyle: "short",
    timeStyle: "short",
  });
}

const statusColour: Record<
  string,
  "amber" | "blue" | "green" | "red" | "gray"
> = {
  pending: "amber",
  processing: "blue",
  completed: "green",
  failed: "red",
  skipped: "gray",
};

export function OCRAssetList({ onRefresh }: { onRefresh?: () => void }) {
  const t = useTranslation();
  const [filter, setFilter] = useState<Filter>("open");
  const [now, setNow] = useState(() => Date.now());

  const { data, error, isLoading } = useAdminOCRAssetList(
    filter === "open" ? undefined : { status: filter },
    { swr: { refreshInterval: REFRESH_MS } },
  );

  useEffect(() => {
    const id = setInterval(() => {
      setNow(Date.now());
      onRefresh?.();
    }, REFRESH_MS);
    return () => clearInterval(id);
  }, [onRefresh]);

  const statusLabel: Record<string, string> = {
    pending: t.ocr.pending,
    processing: t.ocr.processing,
    completed: t.ocr.completed,
    failed: t.ocr.failed,
    skipped: t.ocr.skipped,
  };

  const filters: Array<{ value: Filter; label: string }> = [
    { value: "open", label: "Alle nicht abgeschlossenen" },
    { value: "processing", label: t.ocr.processing },
    { value: "failed", label: t.ocr.failed },
    { value: "skipped", label: t.ocr.skipped },
    { value: "pending", label: t.ocr.pending },
    { value: "completed", label: t.ocr.completed },
  ];

  const assets = sortAssets(data?.assets ?? []);

  return (
    <Box
      p="5"
      borderRadius="lg"
      borderWidth="thin"
      borderColor="border.subtle"
      bgColor="bg.default"
    >
      <styled.h3 fontSize="md" fontWeight="bold" color="fg.default" mb="1">
        Dateien
      </styled.h3>
      <styled.p fontSize="sm" color="fg.muted" mb="4">
        Aktualisiert sich alle 5 Sekunden. Es wird immer nur eine Datei
        gleichzeitig verarbeitet; eine genauere Fortschrittsanzeige liefert die
        OCR-Engine nicht, daher wird die bisherige Laufzeit angezeigt.
      </styled.p>

      <HStack gap="2" flexWrap="wrap" mb="4">
        {filters.map((f) => (
          <Button
            key={f.value}
            type="button"
            size="xs"
            variant={filter === f.value ? "solid" : "outline"}
            onClick={() => setFilter(f.value)}
          >
            {f.label}
          </Button>
        ))}
      </HStack>

      {error ? (
        <styled.p fontSize="sm" color="fg.error">
          Liste konnte nicht geladen werden.
        </styled.p>
      ) : isLoading ? (
        <styled.p fontSize="sm" color="fg.muted">
          Lade…
        </styled.p>
      ) : assets.length === 0 ? (
        <styled.p fontSize="sm" color="fg.muted">
          Keine Dateien in diesem Status.
        </styled.p>
      ) : (
        <Box overflowX="auto">
          <Table.Root size="sm" variant="dense">
            <Table.Head>
              <Table.Row>
                <Table.Header>Datei</Table.Header>
                <Table.Header>Status</Table.Header>
                <Table.Header>Grund</Table.Header>
                <Table.Header>Typ</Table.Header>
                <Table.Header>Größe</Table.Header>
                <Table.Header>Zeit</Table.Header>
              </Table.Row>
            </Table.Head>
            <Table.Body>
              {assets.map((a) => (
                <Table.Row key={a.id}>
                  <Table.Cell maxW="80" wordBreak="break-all">
                    <styled.a
                      href={a.path}
                      target="_blank"
                      rel="noreferrer"
                      color="fg.default"
                      textDecoration="underline"
                    >
                      {displayName(a)}
                    </styled.a>
                  </Table.Cell>
                  <Table.Cell>
                    <Badge
                      size="sm"
                      colorPalette={statusColour[a.status] ?? "gray"}
                    >
                      {statusLabel[a.status] ?? a.status}
                    </Badge>
                  </Table.Cell>
                  <Table.Cell color="fg.muted" maxW="96">
                    <styled.span title={a.error}>
                      {describeReason(a.error)}
                    </styled.span>
                  </Table.Cell>
                  <Table.Cell color="fg.muted">{a.mime_type}</Table.Cell>
                  <Table.Cell color="fg.muted" whiteSpace="nowrap">
                    {formatSize(a.size)}
                  </Table.Cell>
                  <Table.Cell color="fg.muted" whiteSpace="nowrap">
                    {a.processed_at
                      ? a.status === "processing"
                        ? formatElapsed(a.processed_at, now)
                        : formatTime(a.processed_at)
                      : "–"}
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table.Root>
        </Box>
      )}
    </Box>
  );
}

const statusOrder: Record<string, number> = {
  processing: 0,
  failed: 1,
  pending: 2,
  skipped: 3,
  completed: 4,
};

function sortAssets(assets: AdminOCRAssetList200AssetsItem[]) {
  return [...assets].sort(
    (a, b) => (statusOrder[a.status] ?? 9) - (statusOrder[b.status] ?? 9),
  );
}

function displayName(a: AdminOCRAssetList200AssetsItem) {
  return a.filename.startsWith(`${a.id}-`)
    ? a.filename.slice(a.id.length + 1)
    : a.filename;
}

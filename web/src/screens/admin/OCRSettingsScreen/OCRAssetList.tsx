"use client";

import { useEffect, useState } from "react";

import { handle } from "@/api/client";
import {
  adminOCRAssetSkip,
  useAdminOCRAssetList,
} from "@/api/openapi-client/admin";
import { AdminOCRAssetList200AssetsItem } from "@/api/openapi-schema";
import { AdminOCRAssetListStatus } from "@/api/openapi-schema";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { Box, HStack, styled } from "@/styled-system/jsx";

type Filter = "open" | AdminOCRAssetListStatus;

const REFRESH_MS = 5000;

const reasonLabels: Record<string, string> = {
  "asset file is empty": "Datei ist leer (0 Byte)",
  "asset file not found on disk": "Datei fehlt auf dem Server",
  "ocr engine unavailable": "OCR-Engine nicht verfügbar",
  "unsupported mime type": "Dateityp nicht unterstützt",
  "skipped by administrator": "Vom Admin übersprungen",
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

  const { data, error, isLoading, mutate } = useAdminOCRAssetList(
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

  const handleSkip = async (a: AdminOCRAssetList200AssetsItem) => {
    await handle(
      async () => {
        await adminOCRAssetSkip(a.id);
      },
      {
        promiseToast: {
          loading:
            a.status === "processing"
              ? "Breche Verarbeitung ab…"
              : "Überspringe…",
          success: "Übersprungen",
        },
        cleanup: async () => {
          await mutate();
          onRefresh?.();
        },
      },
    );
  };

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
        <styled.ul display="flex" flexDirection="column" gap="2">
          {assets.map((a) => (
            <styled.li
              key={a.id}
              display="flex"
              alignItems="flex-start"
              justifyContent="space-between"
              gap="3"
              px="3"
              py="2"
              borderRadius="md"
              borderWidth="thin"
              borderColor="border.subtle"
              bgColor="bg.subtle"
            >
              <Box minW="0" flex="1">
                <styled.a
                  href={a.path}
                  target="_blank"
                  rel="noreferrer"
                  display="block"
                  fontSize="sm"
                  fontWeight="medium"
                  color="fg.default"
                  overflowWrap="anywhere"
                  _hover={{ textDecoration: "underline" }}
                >
                  {displayName(a)}
                </styled.a>
                {a.error && (
                  <styled.p
                    fontSize="sm"
                    color={a.status === "failed" ? "fg.error" : "fg.muted"}
                    overflowWrap="anywhere"
                    title={a.error}
                  >
                    {describeReason(a.error)}
                  </styled.p>
                )}
                <styled.p fontSize="xs" color="fg.subtle" mt="0.5">
                  {[
                    a.mime_type,
                    formatSize(a.size),
                    a.processed_at
                      ? a.status === "processing"
                        ? formatElapsed(a.processed_at, now)
                        : formatTime(a.processed_at)
                      : undefined,
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </styled.p>
              </Box>
              <HStack gap="2" flexShrink="0">
                {skippable.has(a.status) && (
                  <Button
                    type="button"
                    size="xs"
                    variant="outline"
                    onClick={() => handleSkip(a)}
                  >
                    {a.status === "processing" ? "Abbrechen" : "Überspringen"}
                  </Button>
                )}
                <Badge
                  size="sm"
                  colorPalette={statusColour[a.status] ?? "gray"}
                >
                  {statusLabel[a.status] ?? a.status}
                </Badge>
              </HStack>
            </styled.li>
          ))}
        </styled.ul>
      )}
    </Box>
  );
}

const skippable = new Set(["pending", "processing", "failed"]);

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

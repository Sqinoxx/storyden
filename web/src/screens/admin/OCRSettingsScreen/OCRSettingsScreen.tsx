"use client";

import { useEffect, useState, useCallback } from "react";
import { RefreshCw, Play, FileText, CheckCircle2, Clock, AlertTriangle } from "lucide-react";
import { Box, Flex, Grid, HStack, Stack, styled } from "@/styled-system/jsx";
import { Button } from "@/components/ui/button";
import { fetcher } from "@/api/client";

interface OCRStats {
  total: number;
  pending: number;
  completed: number;
  failed: number;
  skipped: number;
}

export function OCRSettingsScreen() {
  const [stats, setStats] = useState<OCRStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [reindexing, setReindexing] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetcher<OCRStats>({ url: "/admin/ocr/stats" });
      if (data) {
        setStats(data);
      }
    } catch {
      // Ignore network errors
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const handleReindex = async () => {
    setReindexing(true);
    setMessage(null);
    try {
      const data = await fetcher<{ reindexed: number }>({
        url: "/admin/ocr/reindex",
        method: "POST",
      });
      if (data) {
        setMessage(`${data.reindexed} Bild-Assets wurden in die OCR-Warteschlange eingereiht.`);
        fetchStats();
      } else {
        setMessage("Fehler beim Starten der OCR-Stapelverarbeitung.");
      }
    } catch {
      setMessage("Netzwerkfehler beim Anfragen der OCR-Stapelverarbeitung.");
    } finally {
      setReindexing(false);
    }
  };

  return (
    <Stack gap="6" width="full">
      {/* Title & Description */}
      <Box borderBottomWidth="thin" borderColor="border.subtle" pb="4">
        <styled.h2 fontSize="xl" fontWeight="bold" color="fg.default">
          OCR & Bildtext-Erkennung
        </styled.h2>
        <styled.p fontSize="sm" color="fg.muted" mt="1">
          Verwalte die automatische Texterkennung für hochgeladene Bilder und Dokumente. Extrahiert Texte, um Screenshots und Scans über die globale Suche auffindbar zu machen.
        </styled.p>
      </Box>

      {/* Message callout */}
      {message && (
        <Box p="4" borderRadius="md" bgColor="bg.subtle" borderWidth="thin" borderColor="border.default" color="fg.default" fontSize="sm">
          {message}
        </Box>
      )}

      {/* Stats Dashboard Grid */}
      <Grid columns={{ base: 1, sm: 2, md: 4 }} gap="4">
        {/* Completed */}
        <Box p="4" borderRadius="lg" borderWidth="thin" borderColor="border.subtle" bgColor="bg.subtle">
          <Flex alignItems="center" justifyContent="space-between">
            <styled.span fontSize="xs" fontWeight="semibold" color="fg.muted" textTransform="uppercase">
              Erfolgreich erkannt
            </styled.span>
            <CheckCircle2 size={18} style={{ color: "var(--colors-green-500, green)" }} />
          </Flex>
          <styled.div fontSize="2xl" fontWeight="bold" color="fg.default" mt="2">
            {loading ? "…" : (stats?.completed ?? 0)}
          </styled.div>
        </Box>

        {/* Pending */}
        <Box p="4" borderRadius="lg" borderWidth="thin" borderColor="border.subtle" bgColor="bg.subtle">
          <Flex alignItems="center" justifyContent="space-between">
            <styled.span fontSize="xs" fontWeight="semibold" color="fg.muted" textTransform="uppercase">
              Ausstehend
            </styled.span>
            <Clock size={18} style={{ color: "var(--colors-amber-500, orange)" }} />
          </Flex>
          <styled.div fontSize="2xl" fontWeight="bold" color="fg.default" mt="2">
            {loading ? "…" : (stats?.pending ?? 0)}
          </styled.div>
        </Box>

        {/* Failed */}
        <Box p="4" borderRadius="lg" borderWidth="thin" borderColor="border.subtle" bgColor="bg.subtle">
          <Flex alignItems="center" justifyContent="space-between">
            <styled.span fontSize="xs" fontWeight="semibold" color="fg.muted" textTransform="uppercase">
              Fehlgeschlagen
            </styled.span>
            <AlertTriangle size={18} style={{ color: "var(--colors-red-500, red)" }} />
          </Flex>
          <styled.div fontSize="2xl" fontWeight="bold" color="fg.default" mt="2">
            {loading ? "…" : (stats?.failed ?? 0)}
          </styled.div>
        </Box>

        {/* Total Assets */}
        <Box p="4" borderRadius="lg" borderWidth="thin" borderColor="border.subtle" bgColor="bg.subtle">
          <Flex alignItems="center" justifyContent="space-between">
            <styled.span fontSize="xs" fontWeight="semibold" color="fg.muted" textTransform="uppercase">
              Gesamt-Assets
            </styled.span>
            <FileText size={18} />
          </Flex>
          <styled.div fontSize="2xl" fontWeight="bold" color="fg.default" mt="2">
            {loading ? "…" : (stats?.total ?? 0)}
          </styled.div>
        </Box>
      </Grid>

      {/* Control Actions Panel */}
      <Box p="5" borderRadius="lg" borderWidth="thin" borderColor="border.subtle" bgColor="bg.default">
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default" mb="2">
          Verwaltungs-Aktionen
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb="4">
          Stößt die automatische OCR-Texterkennung für alle bestehenden oder unvollständigen Bild-Assets im Hintergrund an.
        </styled.p>

        <HStack gap="3" flexWrap="wrap">
          <Button
            type="button"
            onClick={handleReindex}
            disabled={reindexing}
          >
            <Play size={16} />
            <span>{reindexing ? "OCR wird gestartet..." : "OCR für unvollständige Bilder ausführen"}</span>
          </Button>

          <Button
            type="button"
            variant="outline"
            onClick={fetchStats}
            disabled={loading}
          >
            <RefreshCw size={16} />
            <span>Statistiken aktualisieren</span>
          </Button>
        </HStack>
      </Box>

      {/* Engine Info Box */}
      <Box p="5" borderRadius="lg" borderWidth="thin" borderColor="border.subtle" bgColor="bg.subtle">
        <styled.h3 fontSize="sm" fontWeight="bold" color="fg.default" mb="1">
          ⚙️ OCR-Engine Konfiguration
        </styled.h3>
        <styled.p fontSize="xs" color="fg.muted" lineHeight="relaxed">
          Standardmäßig läuft die Erkennung über <strong>Tesseract OCR</strong> kostenfrei lokal auf deinem Server. Über die Server-Umgebungsvariable <code>OCR_PROVIDER=openai_vision</code> kann optional die OpenAI Vision API für KI-Erkennung aktiviert werden.
        </styled.p>
      </Box>
    </Stack>
  );
}

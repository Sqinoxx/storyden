"use client";

import { useState } from "react";
import {
  RefreshCw,
  Play,
  FileText,
  CheckCircle2,
  Clock,
  AlertTriangle,
  Loader2,
} from "lucide-react";
import { Box, Flex, Grid, HStack, Stack, styled } from "@/styled-system/jsx";
import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import {
  useAdminOCRStats,
  useAdminOCRReindex,
} from "@/api/openapi-client/admin";

export function OCRSettingsScreen() {
  const t = useTranslation();
  const [message, setMessage] = useState<string | null>(null);

  const { data: stats, isLoading: loading, mutate } = useAdminOCRStats();
  const { trigger: reindex, isMutating: reindexing } = useAdminOCRReindex();

  const handleReindex = async () => {
    setMessage(null);
    try {
      const data = await reindex();
      if (data) {
        setMessage(`${data.reindexed} Assets wurden in die OCR-Warteschlange eingereiht.`);
        mutate();
      } else {
        setMessage("Fehler beim Starten der OCR-Stapelverarbeitung.");
      }
    } catch {
      setMessage("Netzwerkfehler beim Anfragen der OCR-Stapelverarbeitung.");
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
          Verwalte die automatische Texterkennung für hochgeladene Bilder und
          PDFs. Extrahiert Texte, um Anhänge über die globale Suche
          auffindbar zu machen.
        </styled.p>
      </Box>

      {/* Message callout */}
      {message && (
        <Box
          p="4"
          borderRadius="md"
          bgColor="bg.subtle"
          borderWidth="thin"
          borderColor="border.default"
          color="fg.default"
          fontSize="sm"
        >
          {message}
        </Box>
      )}

      {/* Stats Dashboard Grid */}
      <Grid columns={{ base: 1, sm: 2, md: 5 }} gap="4">
        <StatCard
          label={t.ocr.completed}
          value={stats?.completed}
          loading={loading}
          icon={<CheckCircle2 size={18} color="var(--colors-green-9)" />}
        />
        <StatCard
          label={t.ocr.pending}
          value={stats?.pending}
          loading={loading}
          icon={<Clock size={18} color="var(--colors-amber-9)" />}
        />
        <StatCard
          label={t.ocr.processing}
          value={stats?.processing}
          loading={loading}
          icon={<Loader2 size={18} color="var(--colors-blue-9)" />}
        />
        <StatCard
          label={t.ocr.failed}
          value={stats?.failed}
          loading={loading}
          icon={<AlertTriangle size={18} color="var(--colors-red-9)" />}
        />
        <StatCard
          label="Gesamt"
          value={stats?.total}
          loading={loading}
          icon={<FileText size={18} />}
        />
      </Grid>

      {/* Control Actions Panel */}
      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default" mb="2">
          Verwaltungs-Aktionen
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb="4">
          Setzt fehlgeschlagene und bereits verarbeitete Assets zurück und
          reiht sie erneut in die Texterkennung ein.
        </styled.p>

        <HStack gap="3" flexWrap="wrap">
          <Button type="button" onClick={handleReindex} disabled={reindexing}>
            <Play size={16} />
            <span>
              {reindexing ? "OCR wird gestartet..." : "OCR erneut ausführen"}
            </span>
          </Button>

          <Button
            type="button"
            variant="outline"
            onClick={() => mutate()}
            disabled={loading}
          >
            <RefreshCw size={16} />
            <span>Statistiken aktualisieren</span>
          </Button>
        </HStack>
      </Box>

      {/* Engine Info Box */}
      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.subtle"
      >
        <styled.h3 fontSize="sm" fontWeight="bold" color="fg.default" mb="1">
          OCR-Engine Konfiguration
        </styled.h3>
        <styled.p fontSize="xs" color="fg.muted" lineHeight="relaxed">
          Digitale PDFs werden ohne externe Abhängigkeiten direkt aus der
          eingebetteten Textebene gelesen. Gescannte PDFs und Bilder werden
          über <strong>Tesseract OCR</strong> lokal auf dem Server erkannt.
          Über die Server-Umgebungsvariable{" "}
          <code>OCR_PROVIDER=openai_vision</code> kann optional die OpenAI
          Vision API für KI-Erkennung aktiviert werden.
        </styled.p>
      </Box>
    </Stack>
  );
}

function StatCard({
  label,
  value,
  loading,
  icon,
}: {
  label: string;
  value: number | undefined;
  loading: boolean;
  icon: React.ReactNode;
}) {
  return (
    <Box
      p="4"
      borderRadius="lg"
      borderWidth="thin"
      borderColor="border.subtle"
      bgColor="bg.subtle"
    >
      <Flex alignItems="center" justifyContent="space-between">
        <styled.span
          fontSize="xs"
          fontWeight="semibold"
          color="fg.muted"
          textTransform="uppercase"
        >
          {label}
        </styled.span>
        {icon}
      </Flex>
      <styled.div fontSize="2xl" fontWeight="bold" color="fg.default" mt="2">
        {loading ? "…" : (value ?? 0)}
      </styled.div>
    </Box>
  );
}

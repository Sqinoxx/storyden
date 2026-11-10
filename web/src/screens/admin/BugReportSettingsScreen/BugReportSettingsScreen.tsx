"use client";

import { formatDate } from "date-fns";
import { useState } from "react";

import { handle } from "@/api/client";
import {
  adminBugReportDelete,
  adminBugReportUpdate,
  useAdminBugReportList,
} from "@/api/openapi-client/bug-reports";
import {
  BugReport,
  BugReportCategory,
  BugReportStatus,
} from "@/api/openapi-schema";
import { MemberBadge } from "@/components/member/MemberBadge/MemberBadge";
import { EmptyState } from "@/components/site/EmptyState";
import { PaginationControls } from "@/components/site/PaginationControls/PaginationControls";
import { UnreadyBanner } from "@/components/site/Unready";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { CardBox, HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { CardBox as cardBox, lstack } from "@/styled-system/patterns";
import { getAssetURL } from "@/utils/asset";

type StatusFilter = BugReportStatus | "all";
type CategoryFilter = BugReportCategory | "all";

const STATUS_FILTERS: { value: StatusFilter; label: string }[] = [
  { value: BugReportStatus.open, label: "Offen" },
  { value: BugReportStatus.planned, label: "Geplant" },
  { value: BugReportStatus.resolved, label: "Erledigt" },
  { value: BugReportStatus.declined, label: "Abgelehnt" },
  { value: "all", label: "Alle" },
];

const CATEGORY_FILTERS: { value: CategoryFilter; label: string }[] = [
  { value: "all", label: "Alle Arten" },
  { value: BugReportCategory.bug, label: "Fehler" },
  { value: BugReportCategory.suggestion, label: "Vorschläge" },
  { value: BugReportCategory.other, label: "Sonstiges" },
];

const STATUS_LABELS: Record<BugReportStatus, string> = {
  open: "Offen",
  planned: "Geplant",
  resolved: "Erledigt",
  declined: "Abgelehnt",
};

const SUGGESTION_STATUS_LABELS: Record<BugReportStatus, string> = {
  ...STATUS_LABELS,
  resolved: "Umgesetzt",
};

const SUGGESTION_ACTIONS: { status: BugReportStatus; label: string }[] = [
  { status: BugReportStatus.open, label: "Wieder öffnen" },
  { status: BugReportStatus.planned, label: "Einplanen" },
  { status: BugReportStatus.resolved, label: "Umgesetzt" },
  { status: BugReportStatus.declined, label: "Ablehnen" },
];

const CATEGORY_LABELS: Record<BugReportCategory, string> = {
  bug: "Fehler",
  suggestion: "Vorschlag",
  other: "Sonstiges",
};

const CATEGORY_COLOURS = {
  bug: "red",
  suggestion: "blue",
  other: "gray",
} as const satisfies Record<BugReportCategory, string>;

export function BugReportSettingsScreen() {
  const [status, setStatus] = useState<StatusFilter>(BugReportStatus.open);
  const [category, setCategory] = useState<CategoryFilter>("all");
  const [page, setPage] = useState(1);
  const [busyId, setBusyId] = useState<string | null>(null);

  const { data, error, mutate } = useAdminBugReportList({
    status: status === "all" ? undefined : status,
    category: category === "all" ? undefined : category,
    page: page.toString(),
  });

  if (!data) {
    return <UnreadyBanner error={error} />;
  }

  function handleStatusFilter(next: StatusFilter) {
    setStatus(next);
    setPage(1);
  }

  function handleCategoryFilter(next: CategoryFilter) {
    setCategory(next);
    setPage(1);
  }

  async function handleSetStatus(report: BugReport, next: BugReportStatus) {
    setBusyId(report.id);
    await handle(
      async () => {
        await adminBugReportUpdate(report.id, { status: next });
        await mutate();
      },
      { cleanup: async () => setBusyId(null) },
    );
  }

  async function handleDelete(report: BugReport) {
    if (!window.confirm("Diese Meldung endgültig löschen?")) return;

    setBusyId(report.id);
    await handle(
      async () => {
        await adminBugReportDelete(report.id);
        await mutate();
      },
      { cleanup: async () => setBusyId(null) },
    );
  }

  return (
    <CardBox className={lstack()} gap="4">
      <WStack>
        <Heading size="md">Feedback</Heading>
        <Button
          type="button"
          size="xs"
          variant="subtle"
          onClick={() => mutate()}
        >
          Aktualisieren
        </Button>
      </WStack>
      <styled.p>
        Fehler und Verbesserungsvorschläge, die Mitglieder über das Kontomenü
        eingereicht haben. Nur für Administratoren sichtbar. Ändert sich der
        Status eines Vorschlags, wird das Mitglied benachrichtigt.
      </styled.p>

      <HStack gap="2" flexWrap="wrap">
        {STATUS_FILTERS.map((f) => (
          <Button
            key={f.value}
            type="button"
            size="xs"
            variant={status === f.value ? "solid" : "outline"}
            onClick={() => handleStatusFilter(f.value)}
          >
            {f.label}
          </Button>
        ))}
      </HStack>

      <HStack gap="2" flexWrap="wrap">
        {CATEGORY_FILTERS.map((f) => (
          <Button
            key={f.value}
            type="button"
            size="xs"
            variant={category === f.value ? "solid" : "outline"}
            onClick={() => handleCategoryFilter(f.value)}
          >
            {f.label}
          </Button>
        ))}
      </HStack>

      {data.bug_reports.length === 0 ? (
        <EmptyState w="full" hideContributionLabel>
          Keine Meldungen vorhanden.
        </EmptyState>
      ) : (
        <>
          <ul className={lstack({ gap: "3" })}>
            {data.bug_reports.map((report) => (
              <BugReportItem
                key={report.id}
                report={report}
                busy={busyId === report.id}
                onSetStatus={handleSetStatus}
                onDelete={handleDelete}
              />
            ))}
          </ul>

          <PaginationControls
            path="/admin"
            currentPage={page}
            totalPages={data.total_pages}
            pageSize={data.page_size}
            onClick={setPage}
          />
        </>
      )}
    </CardBox>
  );
}

function BugReportItem({
  report,
  busy,
  onSetStatus,
  onDelete,
}: {
  report: BugReport;
  busy: boolean;
  onSetStatus: (report: BugReport, status: BugReportStatus) => Promise<void>;
  onDelete: (report: BugReport) => Promise<void>;
}) {
  const isOpen = report.status === BugReportStatus.open;
  const isSuggestion = report.category === BugReportCategory.suggestion;
  const statusLabels = isSuggestion ? SUGGESTION_STATUS_LABELS : STATUS_LABELS;
  const screenshotURL = report.screenshot
    ? getAssetURL(report.screenshot.path)
    : null;

  return (
    <li className={cardBox()}>
      <LStack gap="3">
        <WStack gap="2" alignItems="center" flexWrap="wrap">
          <HStack gap="2" flexWrap="wrap">
            <Badge
              variant="solid"
              colorPalette={CATEGORY_COLOURS[report.category]}
            >
              {CATEGORY_LABELS[report.category]}
            </Badge>
            <Badge variant="outline">{statusLabels[report.status]}</Badge>
            <MemberBadge
              profile={report.author}
              size="sm"
              name="handle"
              as="link"
            />
          </HStack>

          <styled.time fontSize="xs" color="fg.muted">
            {formatDate(report.createdAt, "dd.MM.yyyy HH:mm")}
          </styled.time>
        </WStack>

        <styled.p whiteSpace="pre-wrap" overflowWrap="anywhere">
          {report.description}
        </styled.p>

        {screenshotURL && (
          <styled.a
            href={screenshotURL}
            target="_blank"
            rel="noopener noreferrer"
            w="fit"
          >
            <styled.img
              src={screenshotURL}
              alt="Screenshot"
              maxH="64"
              maxW="full"
              objectFit="contain"
              borderWidth="thin"
              borderColor="border.subtle"
              borderRadius="md"
            />
          </styled.a>
        )}

        <LStack gap="1" fontSize="xs" color="fg.muted">
          {report.page_url && (
            <styled.a
              href={report.page_url}
              target="_blank"
              rel="noopener noreferrer"
              overflowWrap="anywhere"
              textDecoration="underline"
            >
              {report.page_url}
            </styled.a>
          )}
          {report.user_agent && (
            <styled.span overflowWrap="anywhere">
              {report.user_agent}
            </styled.span>
          )}
        </LStack>

        <HStack gap="2" justify="end" flexWrap="wrap">
          <Button
            type="button"
            size="xs"
            variant="subtle"
            colorPalette="red"
            disabled={busy}
            onClick={() => onDelete(report)}
          >
            Löschen
          </Button>
          {isSuggestion ? (
            SUGGESTION_ACTIONS.filter((a) => a.status !== report.status).map(
              (a) => (
                <Button
                  key={a.status}
                  type="button"
                  size="xs"
                  variant="outline"
                  disabled={busy}
                  onClick={() => onSetStatus(report, a.status)}
                >
                  {a.label}
                </Button>
              ),
            )
          ) : (
            <Button
              type="button"
              size="xs"
              variant={isOpen ? "solid" : "outline"}
              loading={busy}
              onClick={() =>
                onSetStatus(
                  report,
                  isOpen ? BugReportStatus.resolved : BugReportStatus.open,
                )
              }
            >
              {isOpen ? "Als erledigt markieren" : "Wieder öffnen"}
            </Button>
          )}
        </HStack>
      </LStack>
    </li>
  );
}

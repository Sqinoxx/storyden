"use client";

import { formatDate } from "date-fns";
import { useState } from "react";

import { useBugReportListOwn } from "@/api/openapi-client/bug-reports";
import {
  BugReport,
  BugReportCategory,
  BugReportStatus,
} from "@/api/openapi-schema";
import { BugReportModal } from "@/components/bug-report/BugReportModal";
import { EmptyState } from "@/components/site/EmptyState";
import { PaginationControls } from "@/components/site/PaginationControls/PaginationControls";
import { UnreadyBanner } from "@/components/site/Unready";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { LightbulbIcon } from "@/components/ui/icons/Lightbulb";
import { useTranslation } from "@/lib/i18n";
import { LStack, WStack, styled } from "@/styled-system/jsx";
import { CardBox as cardBox, lstack } from "@/styled-system/patterns";
import { useDisclosure } from "@/utils/useDisclosure";

const STATUS_COLOURS = {
  open: "gray",
  planned: "blue",
  resolved: "green",
  declined: "red",
} as const satisfies Record<BugReportStatus, string>;

export function MySuggestionsScreen() {
  const t = useTranslation();
  const [page, setPage] = useState(1);
  const modal = useDisclosure();

  const { data, error, mutate } = useBugReportListOwn({
    category: BugReportCategory.suggestion,
    page: page.toString(),
  });

  function handleClose() {
    modal.onClose();
    void mutate();
  }

  return (
    <LStack gap="4">
      <WStack alignItems="center" flexWrap="wrap">
        <Heading>{t.suggestion.pageTitle}</Heading>
        <Button type="button" size="sm" onClick={modal.onOpen}>
          <LightbulbIcon />
          {t.suggestion.newSuggestion}
        </Button>
      </WStack>

      <styled.p color="fg.muted">{t.suggestion.pageDescription}</styled.p>

      {!data ? (
        <UnreadyBanner error={error} />
      ) : data.bug_reports.length === 0 ? (
        <EmptyState w="full" hideContributionLabel>
          {t.suggestion.empty}
        </EmptyState>
      ) : (
        <>
          <ul className={lstack({ gap: "3" })}>
            {data.bug_reports.map((s) => (
              <SuggestionItem key={s.id} suggestion={s} />
            ))}
          </ul>

          <PaginationControls
            path="/suggestions"
            currentPage={page}
            totalPages={data.total_pages}
            pageSize={data.page_size}
            onClick={setPage}
          />
        </>
      )}

      <BugReportModal
        variant="suggestion"
        isOpen={modal.isOpen}
        onOpen={modal.onOpen}
        onClose={handleClose}
      />
    </LStack>
  );
}

function SuggestionItem({ suggestion }: { suggestion: BugReport }) {
  const t = useTranslation();

  return (
    <li className={cardBox()}>
      <LStack gap="2">
        <WStack gap="2" alignItems="center">
          <Badge
            variant="solid"
            colorPalette={STATUS_COLOURS[suggestion.status]}
          >
            {t.suggestion.status[suggestion.status]}
          </Badge>
          <styled.time fontSize="xs" color="fg.muted">
            {formatDate(suggestion.createdAt, "dd.MM.yyyy HH:mm")}
          </styled.time>
        </WStack>
        <styled.p whiteSpace="pre-wrap" overflowWrap="anywhere">
          {suggestion.description}
        </styled.p>
      </LStack>
    </li>
  );
}

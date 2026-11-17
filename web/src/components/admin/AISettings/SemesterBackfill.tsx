"use client";

import Link from "next/link";
import { useState } from "react";

import { handle } from "@/api/client";
import {
  adminSemesterBackfillApply,
  adminSemesterBackfillCancel,
  adminSemesterBackfillStart,
  useAdminSemesterBackfillStatus,
} from "@/api/openapi-client/admin";
import { SemesterBackfillApplyItem } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Heading } from "@/components/ui/heading";
import { useTranslation } from "@/lib/i18n";
import { formatTermKeyLabel } from "@/lib/thread/semester";
import { CardBox, HStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

export function SemesterBackfill() {
  const t = useTranslation();
  const [excluded, setExcluded] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);

  const { data, mutate } = useAdminSemesterBackfillStatus({
    swr: {
      refreshInterval: (latest) => (latest?.running ? 3000 : 0),
    },
  });

  const proposals = data?.proposals ?? [];
  const running = data?.running ?? false;

  const selected: SemesterBackfillApplyItem[] = proposals
    .filter((p) => !excluded.has(p.thread_id))
    .map((p) => ({ thread_id: p.thread_id, term: p.suggested }));

  const toggle = (threadId: string, checked: boolean) => {
    setExcluded((prev) => {
      const next = new Set(prev);
      if (checked) {
        next.delete(threadId);
      } else {
        next.add(threadId);
      }
      return next;
    });
  };

  const start = async () => {
    setBusy(true);
    await handle(
      async () => {
        setExcluded(new Set());
        await mutate(await adminSemesterBackfillStart(), { revalidate: false });
      },
      { cleanup: async () => setBusy(false) },
    );
  };

  const cancel = async () => {
    setBusy(true);
    await handle(
      async () => {
        await mutate(await adminSemesterBackfillCancel(), {
          revalidate: false,
        });
      },
      { cleanup: async () => setBusy(false) },
    );
  };

  const apply = async () => {
    setBusy(true);
    await handle(
      async () => {
        const res = await adminSemesterBackfillApply({ items: selected });
        await mutate(res, { revalidate: false });
        return res;
      },
      {
        promiseToast: {
          loading: t.aiSettings.semesterBackfillApplying,
          success: t.aiSettings.semesterBackfillApplied,
        },
        cleanup: async () => setBusy(false),
      },
    );
  };

  const progress =
    data && (running || data.finished_at)
      ? t.aiSettings.semesterBackfillProgress
          .replace("{checked}", String(data.checked))
          .replace("{total}", String(data.total))
          .replace("{proposals}", String(proposals.length))
      : null;

  return (
    <CardBox className={lstack()}>
      <WStack>
        <Heading size="md">{t.aiSettings.semesterBackfillTitle}</Heading>
        {running ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            loading={busy}
            onClick={cancel}
          >
            {t.aiSettings.tagBackfillStop}
          </Button>
        ) : (
          <Button
            type="button"
            size="sm"
            variant="subtle"
            loading={busy}
            onClick={start}
          >
            {t.aiSettings.semesterBackfillStart}
          </Button>
        )}
      </WStack>

      <styled.p color="fg.muted" fontSize="sm">
        {t.aiSettings.semesterBackfillDescription}
      </styled.p>

      {progress && (
        <styled.p fontSize="sm">
          {progress}
          {data?.error === "cancelled" && ` ${t.aiSettings.tagBackfillStopped}`}
          {data?.error &&
            data.error !== "cancelled" &&
            ` ${t.aiSettings.tagBackfillError}`}
        </styled.p>
      )}

      {proposals.length > 0 && (
        <>
          <styled.ul
            display="flex"
            flexDirection="column"
            gap="2"
            maxH="96"
            overflowY="auto"
            borderWidth="thin"
            borderColor="border.subtle"
            borderRadius="md"
            p="2"
          >
            {proposals.map((p) => (
              <styled.li
                key={p.thread_id}
                display="flex"
                flexDirection="column"
                gap="1"
                pb="2"
                borderBottomWidth="thin"
                borderColor="border.subtle"
                _last={{ borderBottomWidth: "none", pb: "0" }}
              >
                <styled.span fontWeight="medium" fontSize="sm">
                  <Link href={`/t/${p.slug}`} target="_blank">
                    {p.title || p.slug}
                  </Link>
                </styled.span>
                <HStack gap="3" flexWrap="wrap" fontSize="sm">
                  <Checkbox
                    size="sm"
                    checked={!excluded.has(p.thread_id)}
                    onCheckedChange={({ checked }) =>
                      toggle(p.thread_id, checked === true)
                    }
                  >
                    {formatTermKeyLabel(p.current)} (
                    {p.current_explicit
                      ? t.aiSettings.semesterBackfillCurrentExplicit
                      : t.aiSettings.semesterBackfillCurrentImplicit}
                    ) → {formatTermKeyLabel(p.suggested)}
                  </Checkbox>
                  <styled.span color="fg.muted">
                    {t.aiSettings.semesterBackfillEvidence.replace(
                      "{evidence}",
                      p.evidence,
                    )}
                  </styled.span>
                </HStack>
              </styled.li>
            ))}
          </styled.ul>

          <HStack justify="end">
            <Button
              type="button"
              loading={busy}
              disabled={selected.length === 0}
              onClick={apply}
            >
              {t.aiSettings.semesterBackfillApply.replace(
                "{count}",
                String(selected.length),
              )}
            </Button>
          </HStack>
        </>
      )}

      {data?.finished_at && !running && proposals.length === 0 && (
        <styled.p fontSize="sm" color="fg.muted">
          {t.aiSettings.semesterBackfillNothing}
        </styled.p>
      )}
    </CardBox>
  );
}

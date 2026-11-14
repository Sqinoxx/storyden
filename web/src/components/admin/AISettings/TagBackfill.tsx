"use client";

import Link from "next/link";
import { useState } from "react";

import { handle } from "@/api/client";
import {
  adminTagBackfillApply,
  adminTagBackfillCancel,
  adminTagBackfillStart,
  useAdminTagBackfillStatus,
} from "@/api/openapi-client/admin";
import { TagBackfillApplyItem } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Heading } from "@/components/ui/heading";
import { IntelligenceIcon } from "@/components/ui/icons/Intelligence";
import { useTranslation } from "@/lib/i18n";
import { CardBox, HStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

const key = (threadId: string, tag: string) => `${threadId}\u0000${tag}`;

export function TagBackfill(props: { configured: boolean }) {
  const t = useTranslation();
  const [excluded, setExcluded] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);

  const { data, mutate } = useAdminTagBackfillStatus({
    swr: {
      refreshInterval: (latest) => (latest?.running ? 3000 : 0),
    },
  });

  const proposals = data?.proposals ?? [];
  const running = data?.running ?? false;

  const selected: TagBackfillApplyItem[] = proposals
    .map((p) => ({
      thread_id: p.thread_id,
      tags: p.add.filter((tag) => !excluded.has(key(p.thread_id, tag))),
    }))
    .filter((it) => it.tags.length > 0);
  const selectedCount = selected.reduce((n, it) => n + it.tags.length, 0);

  const toggle = (threadId: string, tag: string, checked: boolean) => {
    setExcluded((prev) => {
      const next = new Set(prev);
      if (checked) {
        next.delete(key(threadId, tag));
      } else {
        next.add(key(threadId, tag));
      }
      return next;
    });
  };

  const start = async () => {
    setBusy(true);
    await handle(
      async () => {
        setExcluded(new Set());
        await mutate(await adminTagBackfillStart(), { revalidate: false });
      },
      { cleanup: async () => setBusy(false) },
    );
  };

  const cancel = async () => {
    setBusy(true);
    await handle(
      async () => {
        await mutate(await adminTagBackfillCancel(), { revalidate: false });
      },
      { cleanup: async () => setBusy(false) },
    );
  };

  const apply = async () => {
    setBusy(true);
    await handle(
      async () => {
        const res = await adminTagBackfillApply({ items: selected });
        await mutate(res, { revalidate: false });
        return res;
      },
      {
        promiseToast: {
          loading: t.aiSettings.tagBackfillApplying,
          success: t.aiSettings.tagBackfillApplied,
        },
        cleanup: async () => setBusy(false),
      },
    );
  };

  const progress =
    data && (running || data.finished_at)
      ? t.aiSettings.tagBackfillProgress
          .replace("{checked}", String(data.checked))
          .replace("{total}", String(data.total))
          .replace("{proposals}", String(proposals.length))
      : null;

  return (
    <CardBox className={lstack()}>
      <WStack>
        <Heading size="md">{t.aiSettings.tagBackfillTitle}</Heading>
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
            disabled={!props.configured}
            onClick={start}
          >
            <IntelligenceIcon /> {t.aiSettings.tagBackfillStart}
          </Button>
        )}
      </WStack>

      <styled.p color="fg.muted" fontSize="sm">
        {t.aiSettings.tagBackfillDescription}
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
                  {p.current.length > 0 && (
                    <styled.span color="fg.muted">
                      {t.aiSettings.tagBackfillCurrent}{" "}
                      {p.current.join(", ")}
                    </styled.span>
                  )}
                  {p.add.map((tag) => (
                    <Checkbox
                      key={tag}
                      size="sm"
                      checked={!excluded.has(key(p.thread_id, tag))}
                      onCheckedChange={({ checked }) =>
                        toggle(p.thread_id, tag, checked === true)
                      }
                    >
                      + {tag}
                    </Checkbox>
                  ))}
                </HStack>
              </styled.li>
            ))}
          </styled.ul>

          <HStack justify="end">
            <Button
              type="button"
              loading={busy}
              disabled={selectedCount === 0}
              onClick={apply}
            >
              {t.aiSettings.tagBackfillApply.replace(
                "{count}",
                String(selectedCount),
              )}
            </Button>
          </HStack>
        </>
      )}

      {data?.finished_at && !running && proposals.length === 0 && (
        <styled.p fontSize="sm" color="fg.muted">
          {t.aiSettings.tagBackfillNothing}
        </styled.p>
      )}
    </CardBox>
  );
}

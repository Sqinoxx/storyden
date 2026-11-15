"use client";

import { useQueryState } from "nuqs";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import {
  accessGrantRemove,
  useAccessGrantList,
} from "@/api/openapi-client/access";
import { AccessGrant, AccessPass } from "@/api/openapi-schema";
import { Unready } from "@/components/site/Unready";
import { Button } from "@/components/ui/button";
import { matchesGrantQuery } from "@/lib/access/access";
import { runBulk } from "@/lib/access/bulk";
import { useAccessRevalidate } from "@/lib/access/useAccessRevalidate";
import { useTranslation } from "@/lib/i18n";
import { LStack, styled } from "@/styled-system/jsx";

import { BulkBar, announceBulk, useGrantSelection } from "./BulkBar";
import { ConfirmButton } from "./ConfirmButton";
import { GrantDialog, GrantTarget } from "./GrantDialog";
import { GrantFilters } from "./GrantFilters";
import { GrantRow } from "./GrantRow";

export function useOpenRequests(enabled = true) {
  return useAccessGrantList({ status: "requested" }, { swr: { enabled } });
}

const toTarget = (g: AccessGrant): GrantTarget => ({
  passID: g.pass.id,
  handle: g.account.handle,
});

export function RequestsTab({ passes }: { passes: AccessPass[] }) {
  const t = useTranslation();
  const m = t.access.manage;
  const revalidate = useAccessRevalidate();
  const { data, error } = useOpenRequests();
  const [granting, setGranting] = useState<AccessGrant | undefined>();
  const [bulkGranting, setBulkGranting] = useState<GrantTarget[]>();
  const [passID, setPassID] = useQueryState("pass", { defaultValue: "" });
  const [query, setQuery] = useState("");

  const grants = useMemo(
    () =>
      (data?.grants ?? []).filter(
        (g) => (!passID || g.pass.id === passID) && matchesGrantQuery(g, query),
      ),
    [data, passID, query],
  );

  const selection = useGrantSelection(grants);

  if (!data) return <Unready error={error} />;

  async function reject(g: AccessGrant, reason: string) {
    await handle(async () => {
      await accessGrantRemove(g.pass.id, g.account.handle, {
        action: "reject",
        ...(reason ? { reason } : {}),
      });
      await revalidate();
      toast.success(m.rejected);
    });
  }

  async function rejectSelected(reason: string) {
    const result = await runBulk(selection.selected, (g) =>
      accessGrantRemove(g.pass.id, g.account.handle, {
        action: "reject",
        ...(reason ? { reason } : {}),
      }),
    );
    await revalidate();
    selection.clear();
    announceBulk(t, result);
  }

  const filtering = passID !== "" || query.trim() !== "";
  const count = selection.selected.length;

  return (
    <LStack gap="3" w="full">
      {data.grants.length === 0 ? (
        <styled.p color="fg.muted">{m.noRequests}</styled.p>
      ) : (
        <>
          <GrantFilters
            passes={passes}
            passID={passID}
            onPassChange={(id) => void setPassID(id || null)}
            query={query}
            onQueryChange={setQuery}
          />

          <BulkBar total={grants.length} selection={selection}>
            <Button
              size="sm"
              onClick={() => setBulkGranting(selection.selected.map(toTarget))}
            >
              {m.markPaid}
            </Button>
            <ConfirmButton
              confirmLabel={m.bulkRejectConfirm.replace(
                "{count}",
                String(count),
              )}
              reasonLabel={m.reasonLabel}
              reasonPlaceholder={m.reasonPlaceholder}
              onConfirm={rejectSelected}
            >
              {m.reject}
            </ConfirmButton>
          </BulkBar>

          {grants.length === 0 && filtering ? (
            <styled.p color="fg.muted">{m.noRequestsMatch}</styled.p>
          ) : (
            grants.map((g) => (
              <GrantRow
                key={g.id}
                grant={g}
                selected={selection.isSelected(g.id)}
                onSelectedChange={(c) => selection.toggle(g.id, c)}
                actions={
                  <>
                    <Button size="sm" onClick={() => setGranting(g)}>
                      {m.markPaid}
                    </Button>
                    <ConfirmButton
                      confirmLabel={m.rejectConfirm}
                      reasonLabel={m.reasonLabel}
                      reasonPlaceholder={m.reasonPlaceholder}
                      onConfirm={(reason) => reject(g, reason)}
                    >
                      {m.reject}
                    </ConfirmButton>
                  </>
                }
              />
            ))
          )}
        </>
      )}

      <GrantDialog
        isOpen={granting !== undefined}
        onClose={() => setGranting(undefined)}
        passes={passes}
        passID={granting?.pass.id}
        memberHandle={granting?.account.handle}
        memberName={granting?.account.name}
      />

      <GrantDialog
        isOpen={bulkGranting !== undefined}
        onClose={() => setBulkGranting(undefined)}
        passes={passes}
        targets={bulkGranting}
        onDone={selection.clear}
      />
    </LStack>
  );
}

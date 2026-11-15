"use client";

import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import {
  accessGrantList,
  accessGrantRemove,
  useAccessGrantList,
} from "@/api/openapi-client/access";
import {
  AccessGrant,
  AccessGrantStatus,
  AccessPass,
} from "@/api/openapi-schema";
import { Unready } from "@/components/site/Unready";
import { Button } from "@/components/ui/button";
import { matchesGrantQuery } from "@/lib/access/access";
import { runBulk } from "@/lib/access/bulk";
import { downloadCsv, grantsToCsv } from "@/lib/access/csv";
import { useAccessRevalidate } from "@/lib/access/useAccessRevalidate";
import { useTranslation } from "@/lib/i18n";
import { LStack, styled } from "@/styled-system/jsx";

import { BulkBar, announceBulk, useGrantSelection } from "./BulkBar";
import { ConfirmButton } from "./ConfirmButton";
import { GrantDialog, GrantTarget } from "./GrantDialog";
import { GrantFilters } from "./GrantFilters";
import { GrantRow } from "./GrantRow";

export const GRANT_STATUSES = [
  "active",
  "expired",
  "revoked",
  "rejected",
] as const satisfies AccessGrantStatus[];

export function GrantsTab({ passes }: { passes: AccessPass[] }) {
  const t = useTranslation();
  const m = t.access.manage;
  const revalidate = useAccessRevalidate();

  const [status, setStatus] = useQueryState(
    "status",
    parseAsStringLiteral(GRANT_STATUSES).withDefault("active"),
  );
  const [passID, setPassID] = useQueryState("pass", { defaultValue: "" });
  const [query, setQuery] = useState("");
  const [dialog, setDialog] = useState<{ grant?: AccessGrant } | undefined>();
  const [bulkExtending, setBulkExtending] = useState<GrantTarget[]>();
  const [exporting, setExporting] = useState(false);

  const { data, error } = useAccessGrantList({
    status,
    ...(passID ? { pass_id: passID } : {}),
  });

  const grants = useMemo(
    () => (data?.grants ?? []).filter((g) => matchesGrantQuery(g, query)),
    [data, query],
  );

  const selection = useGrantSelection(grants);
  const selectedActive = selection.selected.filter(
    (g) => g.status === "active",
  );
  const selectedExtendable = selection.selected.filter(
    (g) => g.status === "active" || g.status === "expired",
  );

  async function exportCsv() {
    setExporting(true);
    await handle(
      async () => {
        const all = await accessGrantList(passID ? { pass_id: passID } : {});
        const pass = passes.find((p) => p.id === passID);
        const date = new Date().toISOString().slice(0, 10);
        const name = (pass?.name ?? m.allPasses)
          .replace(/[^\p{L}\p{N}]+/gu, "-")
          .replace(/^-|-$/g, "");
        downloadCsv(
          `${name}-${date}.csv`,
          grantsToCsv(all.grants, m.csv, (s) => t.access.status[s]),
        );
      },
      { cleanup: async () => setExporting(false) },
    );
  }

  async function revoke(g: AccessGrant, reason: string) {
    await handle(async () => {
      await accessGrantRemove(g.pass.id, g.account.handle, {
        action: "revoke",
        ...(reason ? { reason } : {}),
      });
      await revalidate();
      toast.success(m.revoked);
    });
  }

  async function revokeSelected(reason: string) {
    const result = await runBulk(selectedActive, (g) =>
      accessGrantRemove(g.pass.id, g.account.handle, {
        action: "revoke",
        ...(reason ? { reason } : {}),
      }),
    );
    await revalidate();
    selection.clear();
    announceBulk(t, result);
  }

  return (
    <LStack gap="3" w="full">
      <GrantFilters
        passes={passes}
        passID={passID}
        onPassChange={(id) => {
          void setPassID(id || null);
          selection.clear();
        }}
        query={query}
        onQueryChange={setQuery}
        statuses={[...GRANT_STATUSES]}
        status={status}
        onStatusChange={(s) => {
          void setStatus(s as (typeof GRANT_STATUSES)[number]);
          selection.clear();
        }}
      >
        <Button
          size="sm"
          variant="outline"
          loading={exporting}
          onClick={exportCsv}
        >
          {m.exportCsv}
        </Button>
        <Button size="sm" onClick={() => setDialog({})}>
          {m.grantMember}
        </Button>
      </GrantFilters>

      {data && (
        <BulkBar total={grants.length} selection={selection}>
          {selectedExtendable.length > 0 && (
            <Button
              size="sm"
              variant="outline"
              onClick={() =>
                setBulkExtending(
                  selectedExtendable.map((g) => ({
                    passID: g.pass.id,
                    handle: g.account.handle,
                  })),
                )
              }
            >
              {m.extend}
            </Button>
          )}
          {selectedActive.length > 0 && (
            <ConfirmButton
              confirmLabel={m.bulkRevokeConfirm.replace(
                "{count}",
                String(selectedActive.length),
              )}
              reasonLabel={m.reasonLabel}
              reasonPlaceholder={m.reasonPlaceholder}
              onConfirm={revokeSelected}
            >
              {m.revoke}
            </ConfirmButton>
          )}
        </BulkBar>
      )}

      {!data ? (
        <Unready error={error} />
      ) : grants.length === 0 ? (
        <styled.p color="fg.muted">{m.noGrants}</styled.p>
      ) : (
        grants.map((g) => (
          <GrantRow
            key={g.id}
            grant={g}
            showStatus
            selected={selection.isSelected(g.id)}
            onSelectedChange={(c) => selection.toggle(g.id, c)}
            actions={
              <>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setDialog({ grant: g })}
                >
                  {g.status === "active" || g.status === "expired"
                    ? m.extend
                    : m.markPaid}
                </Button>
                {g.status === "active" && (
                  <ConfirmButton
                    confirmLabel={m.revokeConfirm}
                    reasonLabel={m.reasonLabel}
                    reasonPlaceholder={m.reasonPlaceholder}
                    onConfirm={(reason) => revoke(g, reason)}
                  >
                    {m.revoke}
                  </ConfirmButton>
                )}
              </>
            }
          />
        ))
      )}

      <GrantDialog
        isOpen={dialog !== undefined}
        onClose={() => setDialog(undefined)}
        passes={passes}
        passID={dialog?.grant?.pass.id}
        memberHandle={dialog?.grant?.account.handle}
        memberName={dialog?.grant?.account.name}
        note={dialog?.grant?.note}
        defaultChoice={dialog?.grant ? "nextTerm" : "thisTerm"}
      />

      <GrantDialog
        isOpen={bulkExtending !== undefined}
        onClose={() => setBulkExtending(undefined)}
        passes={passes}
        targets={bulkExtending}
        defaultChoice="nextTerm"
        onDone={selection.clear}
      />
    </LStack>
  );
}

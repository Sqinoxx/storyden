"use client";

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
import { Input } from "@/components/ui/input";
import { downloadCsv, grantsToCsv } from "@/lib/access/csv";
import { useAccessRevalidate } from "@/lib/access/useAccessRevalidate";
import { useTranslation } from "@/lib/i18n";
import { Grid, LStack, styled } from "@/styled-system/jsx";

import { ConfirmButton } from "./ConfirmButton";
import { GrantDialog } from "./GrantDialog";
import { GrantRow } from "./GrantRow";

const STATUSES: AccessGrantStatus[] = [
  "active",
  "expired",
  "revoked",
  "rejected",
];

const selectStyle = {
  borderWidth: "thin",
  borderColor: "border.default",
  borderRadius: "sm",
  p: "2",
  bg: "bg.default",
  fontSize: { base: "md", md: "sm" },
  minW: "0",
  w: "full",
} as const;

export function GrantsTab({ passes }: { passes: AccessPass[] }) {
  const t = useTranslation();
  const m = t.access.manage;
  const revalidate = useAccessRevalidate();

  const [status, setStatus] = useState<AccessGrantStatus>("active");
  const [passID, setPassID] = useState("");
  const [query, setQuery] = useState("");
  const [dialog, setDialog] = useState<{ grant?: AccessGrant } | undefined>();
  const [exporting, setExporting] = useState(false);

  const { data, error } = useAccessGrantList({
    status,
    ...(passID ? { pass_id: passID } : {}),
  });

  const grants = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return data?.grants ?? [];
    return (data?.grants ?? []).filter(
      (g) =>
        g.account.name.toLowerCase().includes(q) ||
        g.account.handle.toLowerCase().includes(q),
    );
  }, [data, query]);

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

  return (
    <LStack gap="3" w="full">
      <Grid
        gap="2"
        w="full"
        alignItems="center"
        gridTemplateColumns={{
          base: "repeat(2, minmax(0, 1fr))",
          md: "auto auto minmax(0, 1fr) auto auto",
        }}
      >
        <styled.select
          {...selectStyle}
          aria-label={m.pass}
          value={passID}
          onChange={(e) => setPassID(e.currentTarget.value)}
        >
          <option value="">{m.allPasses}</option>
          {passes.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </styled.select>
        <styled.select
          {...selectStyle}
          aria-label="Status"
          value={status}
          onChange={(e) =>
            setStatus(e.currentTarget.value as AccessGrantStatus)
          }
        >
          {STATUSES.map((s) => (
            <option key={s} value={s}>
              {t.access.status[s]}
            </option>
          ))}
        </styled.select>
        <Input
          size="sm"
          fontSize={{ base: "md", md: "sm" }}
          gridColumn={{ base: "span 2", md: "auto" }}
          placeholder={m.search}
          aria-label={m.search}
          value={query}
          onChange={(e) => setQuery(e.currentTarget.value)}
        />
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
      </Grid>

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
    </LStack>
  );
}

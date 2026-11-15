"use client";

import { parseAsString, useQueryStates } from "nuqs";
import { useMemo, useState } from "react";

import { useAccessGrantList } from "@/api/openapi-client/access";
import { AccessPass } from "@/api/openapi-schema";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PassStats, passStats } from "@/lib/access/access";
import { useIncludedNames } from "@/lib/access/useIncludedNames";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { CardBox } from "@/styled-system/patterns";

import { PassEditor } from "./PassEditor";

const EMPTY: PassStats = { active: 0, requested: 0, expiring: 0, expired: 0 };

export function PassesTab({ passes }: { passes: AccessPass[] }) {
  const t = useTranslation();
  const m = t.access.manage;
  const includedNames = useIncludedNames();
  const { data: all } = useAccessGrantList({});
  const [editing, setEditing] = useState<{ pass?: AccessPass } | undefined>();
  const [, setNav] = useQueryStates({
    tab: parseAsString,
    pass: parseAsString,
    status: parseAsString,
  });

  const stats = useMemo(() => passStats(all?.grants ?? [], new Date()), [all]);

  function show(passID: string, tab: "requests" | "grants", status?: string) {
    void setNav({ tab, pass: passID, status: status ?? null });
  }

  return (
    <LStack gap="3" w="full">
      <Button
        size="sm"
        w={{ base: "full", sm: "auto" }}
        onClick={() => setEditing({})}
      >
        {m.createPass}
      </Button>

      {passes.length === 0 ? (
        <styled.p color="fg.muted">{m.noPassesYet}</styled.p>
      ) : (
        passes.map((p) => {
          const s = stats.get(p.id) ?? EMPTY;
          return (
            <WStack
              key={p.id}
              className={CardBox()}
              p="3"
              gap="3"
              flexDirection={{ base: "column", sm: "row" }}
              alignItems={{ base: "stretch", sm: "center" }}
              aria-label={p.name}
            >
              <LStack gap="1" minW="0" flex="1" overflowWrap="anywhere">
                <HStack gap="2" flexWrap="wrap">
                  <styled.span fontWeight="semibold">{p.name}</styled.span>
                  {p.price_label && (
                    <Badge size="sm" variant="outline">
                      {p.price_label}
                    </Badge>
                  )}
                </HStack>
                <styled.p fontSize="sm" color="fg.muted">
                  {includedNames(p).join(", ")}
                </styled.p>
                {all && (
                  <HStack
                    gap="1"
                    flexWrap="wrap"
                    data-testid="access-pass-stats"
                  >
                    <StatButton onClick={() => show(p.id, "grants", "active")}>
                      {m.statActive.replace("{count}", String(s.active))}
                    </StatButton>
                    <StatButton onClick={() => show(p.id, "requests")}>
                      {m.statRequested.replace("{count}", String(s.requested))}
                    </StatButton>
                    {s.expiring > 0 && (
                      <StatButton
                        warning
                        onClick={() => show(p.id, "grants", "active")}
                      >
                        {m.statExpiring.replace("{count}", String(s.expiring))}
                      </StatButton>
                    )}
                    <StatButton onClick={() => show(p.id, "grants", "expired")}>
                      {m.statExpired.replace("{count}", String(s.expired))}
                    </StatButton>
                  </HStack>
                )}
              </LStack>
              <Button
                size="sm"
                variant="outline"
                w={{ base: "full", sm: "auto" }}
                onClick={() => setEditing({ pass: p })}
              >
                {m.editPass}
              </Button>
            </WStack>
          );
        })
      )}

      <PassEditor
        isOpen={editing !== undefined}
        onClose={() => setEditing(undefined)}
        pass={editing?.pass}
        activeMembers={
          editing?.pass ? stats.get(editing.pass.id)?.active : undefined
        }
      />
    </LStack>
  );
}

function StatButton({
  warning,
  onClick,
  children,
}: {
  warning?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <Button
      size="xs"
      variant="subtle"
      color={warning ? "fg.warning" : undefined}
      onClick={onClick}
    >
      {children}
    </Button>
  );
}

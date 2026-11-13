"use client";

import { useMemo, useState } from "react";

import { useAccessGrantList } from "@/api/openapi-client/access";
import { AccessPass } from "@/api/openapi-schema";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useIncludedNames } from "@/lib/access/useIncludedNames";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { CardBox } from "@/styled-system/patterns";

import { PassEditor } from "./PassEditor";

export function PassesTab({ passes }: { passes: AccessPass[] }) {
  const t = useTranslation();
  const m = t.access.manage;
  const includedNames = useIncludedNames();
  const { data: active } = useAccessGrantList({ status: "active" });
  const [editing, setEditing] = useState<{ pass?: AccessPass } | undefined>();

  const counts = useMemo(() => {
    const c = new Map<string, number>();
    (active?.grants ?? []).forEach((g) =>
      c.set(g.pass.id, (c.get(g.pass.id) ?? 0) + 1),
    );
    return c;
  }, [active]);

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
        passes.map((p) => (
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
                {active && (
                  <Badge size="sm" variant="subtle">
                    {m.members.replace(
                      "{count}",
                      String(counts.get(p.id) ?? 0),
                    )}
                  </Badge>
                )}
              </HStack>
              <styled.p fontSize="sm" color="fg.muted">
                {includedNames(p).join(", ")}
              </styled.p>
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
        ))
      )}

      <PassEditor
        isOpen={editing !== undefined}
        onClose={() => setEditing(undefined)}
        pass={editing?.pass}
      />
    </LStack>
  );
}

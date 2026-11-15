"use client";

import { useMemo, useState } from "react";
import { toast } from "sonner";

import { AccessGrant } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { BulkResult } from "@/lib/access/bulk";
import { Translations, useTranslation } from "@/lib/i18n";
import { HStack, styled } from "@/styled-system/jsx";

export function useGrantSelection(visible: AccessGrant[]) {
  const [ids, setIDs] = useState<Set<string>>(new Set());

  const selected = useMemo(
    () => visible.filter((g) => ids.has(g.id)),
    [visible, ids],
  );

  return {
    selected,
    isSelected: (id: string) => ids.has(id),
    toggle: (id: string, checked: boolean) =>
      setIDs((prev) => {
        const next = new Set(prev);
        if (checked) next.add(id);
        else next.delete(id);
        return next;
      }),
    setAll: (checked: boolean) =>
      setIDs(checked ? new Set(visible.map((g) => g.id)) : new Set()),
    clear: () => setIDs(new Set()),
  };
}

export function announceBulk(t: Translations, { ok, failed }: BulkResult) {
  const m = t.access.manage;
  if (failed > 0) {
    toast.error(
      m.bulkPartial
        .replace("{ok}", String(ok))
        .replace("{failed}", String(failed)),
    );
    return;
  }
  toast.success(m.bulkDone.replace("{count}", String(ok)));
}

export type GrantSelection = ReturnType<typeof useGrantSelection>;

type Props = {
  total: number;
  selection: GrantSelection;
  children?: React.ReactNode;
};

export function BulkBar({ total, selection, children }: Props) {
  const t = useTranslation();
  const m = t.access.manage;
  const count = selection.selected.length;

  if (total === 0) return null;

  const checked =
    count === 0 ? false : count === total ? true : "indeterminate";

  return (
    <HStack
      gap="2"
      w="full"
      flexWrap="wrap"
      px="3"
      py="2"
      borderWidth="thin"
      borderColor="border.subtle"
      borderRadius="md"
      bg={count > 0 ? "bg.subtle" : "transparent"}
      position={count > 0 ? "sticky" : "static"}
      top="0"
      zIndex="docked"
      data-testid="access-bulk-bar"
    >
      <Checkbox
        size="sm"
        checked={checked}
        onCheckedChange={({ checked }) => selection.setAll(checked === true)}
      >
        {count > 0 ? m.selected.replace("{count}", String(count)) : m.selectAll}
      </Checkbox>
      {count > 0 && (
        <>
          <styled.span flex="1" />
          {children}
          <Button size="sm" variant="ghost" onClick={selection.clear}>
            {m.clearSelection}
          </Button>
        </>
      )}
    </HStack>
  );
}

"use client";

import { useState } from "react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import {
  accessGrantRemove,
  useAccessGrantList,
} from "@/api/openapi-client/access";
import { AccessGrant, AccessPass } from "@/api/openapi-schema";
import { Unready } from "@/components/site/Unready";
import { Button } from "@/components/ui/button";
import { useAccessRevalidate } from "@/lib/access/useAccessRevalidate";
import { useTranslation } from "@/lib/i18n";
import { LStack, styled } from "@/styled-system/jsx";

import { ConfirmButton } from "./ConfirmButton";
import { GrantDialog } from "./GrantDialog";
import { GrantRow } from "./GrantRow";

export function useOpenRequests(enabled = true) {
  return useAccessGrantList({ status: "requested" }, { swr: { enabled } });
}

export function RequestsTab({ passes }: { passes: AccessPass[] }) {
  const t = useTranslation();
  const revalidate = useAccessRevalidate();
  const { data, error } = useOpenRequests();
  const [granting, setGranting] = useState<AccessGrant | undefined>();

  if (!data) return <Unready error={error} />;

  async function reject(g: AccessGrant) {
    await handle(async () => {
      await accessGrantRemove(g.pass.id, g.account.handle, {
        action: "reject",
      });
      await revalidate();
      toast.success(t.access.manage.rejected);
    });
  }

  return (
    <LStack gap="2" w="full">
      {data.grants.length === 0 ? (
        <styled.p color="fg.muted">{t.access.manage.noRequests}</styled.p>
      ) : (
        data.grants.map((g) => (
          <GrantRow
            key={g.id}
            grant={g}
            actions={
              <>
                <Button size="sm" onClick={() => setGranting(g)}>
                  {t.access.manage.markPaid}
                </Button>
                <ConfirmButton
                  confirmLabel={t.access.manage.rejectConfirm}
                  onConfirm={() => reject(g)}
                >
                  {t.access.manage.reject}
                </ConfirmButton>
              </>
            }
          />
        ))
      )}

      <GrantDialog
        isOpen={granting !== undefined}
        onClose={() => setGranting(undefined)}
        passes={passes}
        passID={granting?.pass.id}
        memberHandle={granting?.account.handle}
        memberName={granting?.account.name}
      />
    </LStack>
  );
}

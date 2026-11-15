"use client";

import { useAccessGrantHistory } from "@/api/openapi-client/access";
import { AccessGrantHistoryEntry } from "@/api/openapi-schema";
import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { Unready } from "@/components/site/Unready";
import { Badge } from "@/components/ui/badge";
import { useLanguage } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";

import { formatAccessDate } from "./GrantRow";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  handle: string;
  name: string;
  passID?: string;
  requestedAt?: string;
};

export function GrantHistoryDialog(props: Props) {
  if (!props.isOpen) return null;
  return <GrantHistoryContent {...props} />;
}

function GrantHistoryContent({
  isOpen,
  onClose,
  handle,
  name,
  passID,
  requestedAt,
}: Props) {
  const { t, language } = useLanguage();
  const m = t.access.manage;
  const { data, error } = useAccessGrantHistory({
    account_handle: handle,
    ...(passID ? { pass_id: passID } : {}),
  });

  const items: AccessGrantHistoryEntry[] = [
    ...(data?.entries ?? []),
    ...(requestedAt && passID
      ? [
          {
            id: "requested",
            created_at: requestedAt,
            pass_id: passID,
            pass_name: "",
            status: "requested" as const,
          },
        ]
      : []),
  ].sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at));

  return (
    <ModalDrawer
      isOpen={isOpen}
      onClose={onClose}
      title={m.historyTitle.replace("{name}", name)}
    >
      {!data ? (
        <Unready error={error} />
      ) : (
        <LStack gap="3" w="full" data-testid="access-history">
          {items.length === 0 && (
            <styled.p color="fg.muted">{m.historyEmpty}</styled.p>
          )}
          {items.map((e) => (
            <LStack
              key={e.id}
              gap="1"
              w="full"
              pl="3"
              borderLeftWidth="medium"
              borderColor="border.default"
              data-testid="access-history-entry"
            >
              <HStack gap="2" flexWrap="wrap" fontSize="sm">
                <styled.span fontWeight="semibold">
                  {formatAccessDate(e.created_at, language)}
                </styled.span>
                <Badge size="sm" variant="subtle">
                  {t.access.status[e.status]}
                </Badge>
                {!passID && e.pass_name && (
                  <styled.span>{e.pass_name}</styled.span>
                )}
              </HStack>
              <HStack gap="2" flexWrap="wrap" fontSize="sm" color="fg.muted">
                {e.expires_at && (
                  <styled.span>
                    {m.untilDate.replace(
                      "{date}",
                      formatAccessDate(e.expires_at, language),
                    )}
                  </styled.span>
                )}
                {e.decided_by && (
                  <styled.span>
                    {m.decidedBy.replace("{name}", e.decided_by.name)}
                  </styled.span>
                )}
              </HStack>
              {e.note && (
                <styled.p fontSize="sm" color="fg.subtle">
                  {e.note}
                </styled.p>
              )}
              {e.reason && (
                <styled.p fontSize="sm" color="fg.warning">
                  {t.access.reason.replace("{reason}", e.reason)}
                </styled.p>
              )}
            </LStack>
          ))}
        </LStack>
      )}
    </ModalDrawer>
  );
}

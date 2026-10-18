"use client";

import { Portal } from "@ark-ui/react";

import * as Tooltip from "@/components/ui/tooltip";
import { useTranslation } from "@/lib/i18n";
import { usePresence } from "@/lib/presence/usePresence";
import { HStack, styled } from "@/styled-system/jsx";

import styles from "./PresenceIndicator.module.css";

type Props = {
  compact?: boolean;
};

export function PresenceIndicator({ compact }: Props) {
  const t = useTranslation();
  const presence = usePresence();

  if (!presence) {
    return null;
  }

  const { online, inCategory, category } = presence;
  const showCategory = category !== undefined && inCategory !== undefined;

  const onlineLabel = t.presence.online.replace("{count}", String(online));
  const description = [
    t.presence.onlineDescription.replace("{count}", String(online)),
    showCategory &&
      (inCategory > 0
        ? t.presence.inCategoryDescription
            .replace("{count}", String(inCategory))
            .replace("{name}", category.name)
        : t.presence.inCategoryAlone.replace("{name}", category.name)),
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <Tooltip.Root openDelay={300} positioning={{ placement: "bottom" }}>
      <Tooltip.Trigger asChild>
        <HStack
          data-testid="presence-indicator"
          aria-label={description}
          gap="1.5"
          px={compact ? "2" : "2.5"}
          h={compact ? "8" : "7"}
          flexShrink="0"
          borderRadius="full"
          borderWidth="thin"
          borderColor="border.subtle"
          bg="bg.subtle/60"
          color="fg.muted"
          fontSize="xs"
          fontWeight="medium"
          fontVariantNumeric="tabular-nums"
          whiteSpace="nowrap"
          cursor="default"
          userSelect="none"
          transition="colors"
          _hover={{ color: "fg.default", borderColor: "border.default" }}
        >
          <span className={styles["dot"]} />
          <styled.span>{compact ? online : onlineLabel}</styled.span>
          {showCategory && inCategory > 0 && (
            <>
              <styled.span
                h="3"
                borderLeftWidth="thin"
                borderColor="border.default"
                display={compact ? "none" : "inline-block"}
              />
              <styled.span
                display={compact ? "none" : "inline"}
                color="fg.default"
                data-testid="presence-in-category"
              >
                {t.presence.inCategory.replace("{count}", String(inCategory))}
              </styled.span>
            </>
          )}
        </HStack>
      </Tooltip.Trigger>
      <Portal>
        <Tooltip.Positioner>
          <Tooltip.Arrow>
            <Tooltip.ArrowTip />
          </Tooltip.Arrow>
          <Tooltip.Content p="2" borderRadius="lg" maxW="xs" fontSize="xs">
            {description}
          </Tooltip.Content>
        </Tooltip.Positioner>
      </Portal>
    </Tooltip.Root>
  );
}

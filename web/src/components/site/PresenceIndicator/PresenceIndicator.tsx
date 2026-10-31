"use client";

import { Portal } from "@ark-ui/react";

import * as Tooltip from "@/components/ui/tooltip";
import { useTranslation } from "@/lib/i18n";
import { usePresence } from "@/lib/presence/usePresence";
import { HStack, styled } from "@/styled-system/jsx";

import styles from "./PresenceIndicator.module.css";

function usePresenceLabels() {
  const t = useTranslation();
  const presence = usePresence();

  if (!presence) {
    return undefined;
  }

  const { online, inCategory, category } = presence;
  const showCategory = category !== undefined && inCategory !== undefined;

  const onlineLabel = t.presence.online.replace("{count}", String(online));
  const inCategoryLabel =
    showCategory && inCategory > 0
      ? t.presence.inCategory.replace("{count}", String(inCategory))
      : undefined;
  const description = [
    (online === 1
      ? t.presence.onlineDescriptionOne
      : t.presence.onlineDescription
    ).replace("{count}", String(online)),
    showCategory &&
      (inCategory > 0
        ? (inCategory === 1
            ? t.presence.inCategoryDescriptionOne
            : t.presence.inCategoryDescription
          )
            .replace("{count}", String(inCategory))
            .replace("{name}", category.name)
        : t.presence.inCategoryAlone.replace("{name}", category.name)),
  ]
    .filter(Boolean)
    .join(" ");

  return { onlineLabel, inCategoryLabel, description };
}

export function PresenceIndicator() {
  const labels = usePresenceLabels();
  if (!labels) {
    return null;
  }

  const { onlineLabel, inCategoryLabel, description } = labels;

  return (
    <Tooltip.Root openDelay={300} positioning={{ placement: "bottom" }}>
      <Tooltip.Trigger asChild>
        <HStack
          data-testid="presence-indicator"
          aria-label={description}
          gap="1.5"
          px="2.5"
          h="7"
          minW="0"
          overflow="hidden"
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
          <styled.span minW="0" overflow="hidden" textOverflow="ellipsis">
            {onlineLabel}
          </styled.span>
          {inCategoryLabel && (
            <>
              <styled.span
                h="3"
                borderLeftWidth="thin"
                borderColor="border.default"
              />
              <styled.span color="fg.default" data-testid="presence-in-category">
                {inCategoryLabel}
              </styled.span>
            </>
          )}
        </HStack>
      </Tooltip.Trigger>
      <Portal>
        <Tooltip.Positioner>
          <Tooltip.Content p="2" borderRadius="lg" maxW="xs" fontSize="xs">
            {description}
          </Tooltip.Content>
          <Tooltip.Arrow>
            <Tooltip.ArrowTip />
          </Tooltip.Arrow>
        </Tooltip.Positioner>
      </Portal>
    </Tooltip.Root>
  );
}

export function PresenceFooter() {
  const labels = usePresenceLabels();
  if (!labels) {
    return null;
  }

  const { onlineLabel, inCategoryLabel, description } = labels;

  return (
    <HStack
      data-testid="presence-indicator"
      aria-label={description}
      w="full"
      justify="center"
      gap="1.5"
      pt="1.5"
      borderTopWidth="thin"
      borderColor="border.muted"
      color="fg.muted"
      fontSize="xs"
      fontVariantNumeric="tabular-nums"
      whiteSpace="nowrap"
      userSelect="none"
    >
      <span className={styles["dot"]} />
      <span>{onlineLabel}</span>
      {inCategoryLabel && (
        <>
          <span>·</span>
          <styled.span color="fg.default" data-testid="presence-in-category">
            {inCategoryLabel}
          </styled.span>
        </>
      )}
    </HStack>
  );
}

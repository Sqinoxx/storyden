import Link from "next/link";
import { PropsWithChildren } from "react";

import { css, cx } from "@/styled-system/css";
import { HStack, styled } from "@/styled-system/jsx";
import { button } from "@/styled-system/recipes";
import { ChevronRightIcon } from "@/components/ui/icons/Chevron";

type Props = {
  href: string;
  controls?: React.ReactNode;
  size?: "xs" | "sm" | "md" | "lg";
  collapsible?: boolean;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
};

export function NavigationHeader({
  children,
  href,
  controls,
  size = "xs",
  collapsible = false,
  isCollapsed = false,
  onToggleCollapse,
}: PropsWithChildren<Props>) {
  const buttonSize = size === "lg" ? "md" : size === "md" ? "sm" : size === "sm" ? "sm" : "xs";
  const fontSize = size;
  const fontWeight = size === "xs" ? "medium" : size === "sm" ? "semibold" : "bold";

  const linkStyles = cx(
    button({ variant: "ghost", size: buttonSize }),
    css({
      p: size === "md" || size === "lg" ? "1.5" : "1",
    }),
  );

  return (
    <HStack w="full" justifyContent="space-between" alignItems="center" gap="1">
      <HStack gap="0.5" alignItems="center" minW="0" flex="1">
        {collapsible && (
          <button
            type="button"
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              onToggleCollapse?.();
            }}
            className={css({
              display: "inline-flex",
              alignItems: "center",
              justifyContent: "center",
              p: "1",
              borderRadius: "sm",
              cursor: "pointer",
              color: "fg.subtle",
              flexShrink: "0",
              _hover: { color: "fg.default", bg: "bg.muted" },
            })}
            aria-label={isCollapsed ? "Expand section" : "Collapse section"}
          >
            <ChevronRightIcon
              className={css({
                w: "4",
                h: "4",
              })}
              style={{
                transition: "transform 0.15s ease",
                transform: isCollapsed ? "rotate(0deg)" : "rotate(90deg)",
              }}
            />
          </button>
        )}

        <Link className={linkStyles} href={href}>
          <styled.h1 fontSize={fontSize} fontWeight={fontWeight}>
            {children}
          </styled.h1>
        </Link>
      </HStack>

      {controls && (
        <HStack flexShrink="0">
          {controls}
        </HStack>
      )}
    </HStack>
  );
}

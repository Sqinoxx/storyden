"use client";

import Link from "next/link";
import { Children, Fragment, PropsWithChildren, useLayoutEffect, useRef } from "react";

import { BreadcrumbIcon } from "@/components/ui/icons/Breadcrumb";
import { styled } from "@/styled-system/jsx";

export function BreadcrumbTrail({ children }: PropsWithChildren) {
  const ref = useRef<HTMLElement>(null);
  const items = Children.toArray(children);

  useLayoutEffect(() => {
    const el = ref.current;
    if (el) el.scrollLeft = el.scrollWidth;
  });

  return (
    <styled.nav
      ref={ref}
      aria-label="Breadcrumb"
      minW="0"
      flex="1"
      overflowX="auto"
      style={{ scrollbarWidth: "none" }}
    >
      <styled.ol
        display="flex"
        alignItems="center"
        gap="1"
        w="max"
        fontSize="sm"
        color="fg.muted"
      >
        {items.map((item, i) => (
          <Fragment key={i}>
            {i > 0 && (
              <styled.li aria-hidden="true" display="flex" color="fg.subtle">
                <BreadcrumbIcon height="3.5" width="3.5" />
              </styled.li>
            )}
            <styled.li display="flex" minW="0">
              {item}
            </styled.li>
          </Fragment>
        ))}
      </styled.ol>
    </styled.nav>
  );
}

type BreadcrumbLinkProps = PropsWithChildren<{
  href: string;
  current?: boolean;
}>;

export function BreadcrumbLink({ href, current, children }: BreadcrumbLinkProps) {
  return (
    <styled.span
      display="block"
      maxW="48"
      overflow="hidden"
      textOverflow="ellipsis"
      whiteSpace="nowrap"
      fontWeight={current ? "medium" : "normal"}
      color={current ? "fg.default" : "fg.muted"}
      _hover={{ color: "fg.default" }}
      transition="colors"
    >
      <Link href={href} aria-current={current ? "page" : undefined}>
        {children}
      </Link>
    </styled.span>
  );
}

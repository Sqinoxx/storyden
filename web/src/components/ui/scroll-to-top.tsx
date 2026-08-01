"use client";

import Link from "next/link";
import { ReactNode, useCallback } from "react";

import { useTranslation } from "@/lib/i18n";
import { button } from "@/styled-system/recipes";
import { scrollToTop } from "@/utils/scroll";

interface ScrollToTopProps {
  children?: ReactNode;
}

export function ScrollToTop({ children }: ScrollToTopProps) {
  const t = useTranslation();
  const handleClick = useCallback((e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    scrollToTop();
  }, []);

  return (
    <Link
      href="#"
      className={button({
        variant: "subtle",
        size: "xs",
      })}
      onClick={handleClick}
    >
      {children ?? t.common?.scrollToTop ?? "scroll to top"}
    </Link>
  );
}

import Link from "next/link";
import { PropsWithChildren } from "react";

import { css, cx } from "@/styled-system/css";
import { HStack, WStack, styled } from "@/styled-system/jsx";
import { button } from "@/styled-system/recipes";

type Props = {
  href: string;
  controls?: React.ReactNode;
  size?: "xs" | "sm" | "md" | "lg";
};

export function NavigationHeader({
  children,
  href,
  controls,
  size = "xs",
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
    <WStack>
      <Link className={linkStyles} href={href}>
        <styled.h1 fontSize={fontSize} fontWeight={fontWeight}>
          {children}
        </styled.h1>
      </Link>

      {controls}
    </WStack>
  );
}

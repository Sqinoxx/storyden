"use client";

import { usePathname } from "next/navigation";

import { BackAction } from "@/components/site/Action/Back";
import { WStack } from "@/styled-system/jsx";

export function BareBackAction() {
  const pathname = usePathname();

  if (pathname === "/login") {
    return null;
  }

  return (
    <WStack>
      <BackAction />
    </WStack>
  );
}

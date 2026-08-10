"use client";

import { usePathname } from "next/navigation";
import { useEffect, useState } from "react";

import { useSession } from "@/auth";

export function useMobileCommandBar() {
  const pathname = usePathname();
  const [isExpanded, setExpanded] = useState(false);
  const [isAccountMenuOpen, setAccountMenuOpen] = useState(false);
  const account = useSession();

  // Close the menu for either navigation events or outside clicks/taps:

  useEffect(() => {
    setExpanded(false);
    setAccountMenuOpen(false);
  }, [pathname]);

  function onExpand() {
    setAccountMenuOpen(false);
    setExpanded(true);
  }

  function onClose() {
    setExpanded(false);
  }

  function onAccountMenuOpenChange(open: boolean) {
    setAccountMenuOpen(open);
    if (open) {
      setExpanded(false);
    }
  }

  return {
    isExpanded,
    onExpand,
    onClose,
    account,
    isAccountMenuOpen,
    onAccountMenuOpenChange,
  };
}

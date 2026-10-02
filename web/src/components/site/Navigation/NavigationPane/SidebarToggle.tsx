"use client";

import { Sidebar } from "@/components/graphics/Sidebar/Sidebar";

import { Button } from "@/components/ui/button";

import { useSidebar } from "./useSidebar";
import { useTranslation } from "@/lib/i18n";

type Props = {
  initialValue: boolean;
};

export function SidebarToggle({ initialValue }: Props) {
  const t = useTranslation();
  const { setShowLeftBar, showLeftBar } = useSidebar(initialValue);
  const isOpen = showLeftBar;
  const label = isOpen
    ? t.nav.closeSidebar
    : t.nav.openSidebar;

  return (
    <Button
      type="button"
      size="md"
      p="0"
      variant="ghost"
      onClick={setShowLeftBar}
      aria-label={label}
      title={label}
      aria-expanded={isOpen}
      aria-controls="navigation__leftbar navigation__rightbar"
      data-state={isOpen ? "open" : "closed"}
    >
      <Sidebar open={isOpen} aria-hidden="true" focusable="false" />
    </Button>
  );
}

"use client";

import { useEffect, useState } from "react";

import { CreatePageAction } from "@/components/library/CreatePage";
import { LibraryPageTree } from "@/components/library/LibraryPageTree/LibraryPageTree";
import { LibraryIcon } from "@/components/ui/icons/Library";
import { HStack, LStack } from "@/styled-system/jsx";

import { LibraryLabel, LibraryRoute } from "../Anchors/Library";
import { NavigationHeader } from "../ContentNavigationList/NavigationHeader";

import { Props, useLibraryNavigationTree } from "./useLibraryNavigationTree";
import { useTranslation } from "@/lib/i18n";

export function LibraryNavigationTree(props: Props) {
  const { ready, data, canManageLibrary } = useLibraryNavigationTree(props);
  const t = useTranslation();
  const [isCollapsed, setIsCollapsed] = useState(false);

  useEffect(() => {
    try {
      const stored = localStorage.getItem("nav_section_collapsed_library");
      if (stored !== null) {
        setIsCollapsed(stored === "true");
      }
    } catch { }
  }, []);

  const handleToggleCollapse = () => {
    setIsCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem("nav_section_collapsed_library", String(next));
      } catch { }
      return next;
    });
  };

  if (!ready) {
    // TODO: Render a small version of <Unready /> that's more suitable for this
    return null;
  }

  const { currentNode } = props;

  return (
    <LStack gap="1">
      <NavigationHeader
        href={LibraryRoute}
        size="sm"
        controls={
          canManageLibrary && <CreatePageAction variant="ghost" hideLabel />
        }
        collapsible
        isCollapsed={isCollapsed}
        onToggleCollapse={handleToggleCollapse}
      >
        <HStack gap="1">
          <LibraryIcon />
          {t.nav.library}
        </HStack>
      </NavigationHeader>

      {!isCollapsed && (
        <LibraryPageTree
          currentNode={currentNode}
          nodes={data.nodes}
          canManageLibrary={canManageLibrary}
        />
      )}
    </LStack>
  );
}

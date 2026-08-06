"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useSearchParams } from "next/navigation";

import { useNodeDraftList, useNodeList } from "@/api/openapi-client/nodes";
import { useThreadList } from "@/api/openapi-client/threads";
import { Visibility } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { NavigationHeader } from "@/components/site/Navigation/ContentNavigationList/NavigationHeader";
import { BulletIcon } from "@/components/ui/icons/Bullet";
import { DraftIcon } from "@/components/ui/icons/Draft";
import { useTranslation } from "@/lib/i18n";
import { css } from "@/styled-system/css";
import { HStack, LStack } from "@/styled-system/jsx";

export function DraftsSidebarSection() {
  const session = useSession();
  const t = useTranslation();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [isCollapsed, setIsCollapsed] = useState(false);

  useEffect(() => {
    try {
      const stored = localStorage.getItem("nav_section_collapsed_drafts");
      if (stored !== null) {
        setIsCollapsed(stored === "true");
      }
    } catch {}
  }, []);

  const handleToggleCollapse = () => {
    setIsCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem("nav_section_collapsed_drafts", String(next));
      } catch {}
      return next;
    });
  };

  const handle = session?.handle;

  const { data: threadsData } = useThreadList(
    handle
      ? {
          author: handle,
          visibility: [Visibility.draft, Visibility.review],
        }
      : undefined,
    { swr: { enabled: !!handle } },
  );

  const { data: nodesData } = useNodeList(
    handle
      ? {
          author: handle,
          visibility: [Visibility.draft, Visibility.review],
        }
      : undefined,
    { swr: { enabled: !!handle } },
  );

  const { data: nodeDraftsData } = useNodeDraftList(undefined, {
    swr: { enabled: !!handle },
  });

  if (!session || !handle) {
    return null;
  }

  const threadDrafts = threadsData?.threads ?? [];
  const nodeDrafts = nodesData?.nodes ?? [];
  const versionDrafts = (nodeDraftsData?.drafts ?? []).filter(
    (d) => d.author?.handle === handle,
  );

  const totalDraftCount =
    threadDrafts.length + nodeDrafts.length + versionDrafts.length;

  if (totalDraftCount === 0) {
    return null;
  }

  const itemLinkStyles = css({
    display: "flex",
    alignItems: "center",
    gap: "2",
    w: "full",
    px: "2",
    py: "1",
    fontSize: "xs",
    borderRadius: "md",
    color: "fg.default",
    _hover: {
      bg: "bg.muted",
    },
  });

  return (
    <LStack gap="1" w="full" mt="2">
      <NavigationHeader
        href="/drafts"
        size="sm"
        collapsible
        isCollapsed={isCollapsed}
        onToggleCollapse={handleToggleCollapse}
      >
        <HStack gap="2">
          <DraftIcon width="4" height="4" />
          {t.nav?.drafts ?? "Entwürfe"} ({totalDraftCount})
        </HStack>
      </NavigationHeader>

      {!isCollapsed && (
        <LStack gap="0.5" pl="5" w="full">
          {threadDrafts.map((thread) => {
            const href = `/new?id=${thread.id}`;
            const isSelected =
              pathname === "/new" && searchParams?.get("id") === thread.id;
            return (
              <Link
                key={thread.id}
                href={href}
                className={itemLinkStyles}
                style={{
                  background: isSelected
                    ? "var(--colors-bg-selected)"
                    : undefined,
                }}
              >
                <BulletIcon />
                <span
                  style={{
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {thread.title || "Unbenannter Entwurf"}
                </span>
              </Link>
            );
          })}

          {nodeDrafts.map((node) => {
            const href = `/l/${node.slug}`;
            const isSelected = pathname === href;
            return (
              <Link
                key={node.id}
                href={href}
                className={itemLinkStyles}
                style={{
                  background: isSelected
                    ? "var(--colors-bg-selected)"
                    : undefined,
                }}
              >
                <BulletIcon />
                <span
                  style={{
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {node.name || "Unbenannte Seite"}
                </span>
              </Link>
            );
          })}

          {versionDrafts.map((draft) => {
            const href = `/l/${draft.node.slug}?version=${draft.id}`;
            const isSelected =
              pathname === `/l/${draft.node.slug}` &&
              searchParams?.get("version") === draft.id;
            return (
              <Link
                key={draft.id}
                href={href}
                className={itemLinkStyles}
                style={{
                  background: isSelected
                    ? "var(--colors-bg-selected)"
                    : undefined,
                }}
              >
                <BulletIcon />
                <span
                  style={{
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {draft.name || draft.node.name || "Unbenannte Version"}
                </span>
              </Link>
            );
          })}
        </LStack>
      )}
    </LStack>
  );
}


"use client";

import { useState } from "react";
import { ChevronDown, ChevronUp, Download } from "lucide-react";

import { CategoryBadge } from "@/components/category/CategoryBadge";
import {
  FileAttachmentBadge,
} from "@/components/post/FileAttachmentList";
import { extractDocumentAssetsFromThread } from "@/components/post/ThreadCard";
import { Unready } from "@/components/site/Unready";
import { DiscussionIcon } from "@/components/ui/icons/Discussion";
import { SlugIcon } from "@/components/ui/icons/Slug";
import * as Table from "@/components/ui/table";
import { Asset, ThreadReference } from "@/api/openapi-schema";
import { css, cva } from "@/styled-system/css";
import { Box, HStack, LStack, styled } from "@/styled-system/jsx";
import { Floating } from "@/styled-system/patterns";
import { ScrollToTop } from "@/components/ui/scroll-to-top";
import { useLanguage } from "@/lib/i18n";

import { Props, useCategoryScreen } from "./CategoryScreen";

const valueStyles = cva({
  base: {},
  defaultVariants: {
    style: "base",
  },
  variants: {
    style: {
      base: {},
      numeric: {
        fontVariant: "tabular-nums",
        fontFamily: "mono",
      },
    },
  },
});

/**
 * Collects all unique document assets from every thread on the current page,
 * using the same extraction logic as the ThreadCard component so that files
 * embedded in body HTML are also captured.
 */
function collectAllDocumentAssets(threads: ThreadReference[]): Asset[] {
  const seen = new Set<string>();
  const result: Asset[] = [];

  for (const thread of threads) {
    const assets = extractDocumentAssetsFromThread(thread);
    for (const asset of assets) {
      const key = asset.id ?? asset.path ?? asset.filename;
      if (key && seen.has(key)) continue;
      if (key) seen.add(key);
      result.push(asset);
    }
  }

  return result;
}

export function CategoryScreenContextPane(props: Props) {
  const { ready, error, data } = useCategoryScreen(props);
  const { t } = useLanguage();
  const [filesOpen, setFilesOpen] = useState(true);

  if (!ready) {
    return <Unready error={error} />;
  }

  const { category } = data;

  const tableData = [
    {
      label: "slug",
      icon: SlugIcon,
      value: category.slug,
      style: "numeric" as const,
    },
    {
      label: category.postCount === 1 ? "thread" : "threads",
      icon: DiscussionIcon,
      value: `${category.postCount}`,
    },
  ];

  // Collect all document assets from every thread visible on this page,
  // using the same body-parsing logic as the ThreadCard.
  const threads = (props.initialThreadList?.threads ?? []) as ThreadReference[];
  const documentAssets = collectAllDocumentAssets(threads);

  return (
    <LStack gap="3" w="full" height="full">
      {/* ── Box 1: Category Info & Stats ── */}
      <Box className={Floating()} borderRadius="md" w="full" p="3">
        <LStack gap="2">
          <CategoryBadge category={category} />
          {category.description && (
            <p className={css({ color: "fg.muted", fontSize: "sm" })}>
              {category.description}
            </p>
          )}

          <Table.Root size="sm">
            <Table.Body>
              {tableData.map((item) => (
                <Table.Row key={item.label}>
                  <Table.Cell fontWeight="medium" color="fg.muted">
                    <HStack gap="1">
                      <item.icon width="4" />
                      <span>{item.label}</span>
                    </HStack>
                  </Table.Cell>
                  <Table.Cell
                    className={valueStyles({ style: item.style })}
                    display="flex"
                    justifyContent="flex-end"
                    alignItems="center"
                    textAlign="right"
                  >
                    {item.value}
                  </Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table.Root>

          <Box pt="1">
            <ScrollToTop />
          </Box>
        </LStack>
      </Box>

      {/* ── Box 2: Dateien in diesem Fach (Eigenständige Box - unten ausgerichtet) ── */}
      {documentAssets.length > 0 && (
        <Box className={Floating()} borderRadius="md" w="full" p="3" mt="auto">
          {/* Header row – click to toggle */}
          <styled.button
            type="button"
            display="flex"
            alignItems="center"
            justifyContent="space-between"
            w="full"
            cursor="pointer"
            mb={filesOpen ? "2.5" : "0"}
            onClick={() => setFilesOpen((o) => !o)}
          >
            <styled.span
              fontSize="sm"
              fontWeight="semibold"
              color="fg.default"
              display="flex"
              alignItems="center"
              gap="2"
            >
              <Download size={14} />
              {t.category.filesInThisCategory}
              <styled.span fontSize="xs" fontWeight="normal" color="fg.muted">
                ({documentAssets.length})
              </styled.span>
            </styled.span>
            <styled.span color="fg.muted">
              {filesOpen ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
            </styled.span>
          </styled.button>

          {/* Standalone File List */}
          {filesOpen && (
            <styled.div display="flex" flexDirection="column" gap="1.5">
              {documentAssets.map((asset) => (
                <FileAttachmentBadge
                  key={asset.id ?? asset.filename}
                  asset={asset}
                />
              ))}
            </styled.div>
          )}
        </Box>
      )}
    </LStack>
  );
}

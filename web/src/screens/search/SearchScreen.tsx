"use client";

import { SlidersHorizontalIcon } from "lucide-react";
import { useState } from "react";

import { DatagraphSearchResults } from "@/components/search/DatagraphSearchResults";
import { UnreadyBanner } from "@/components/site/Unready";

import { DatagraphItemKind } from "@/api/openapi-schema";
import { PaginationControls } from "@/components/site/PaginationControls/PaginationControls";
import { MultiSelectPicker } from "@/components/ui/MultiSelectPicker";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { DatagraphKindFilterField } from "@/components/ui/form/DatagraphKindFilterField";
import { CancelIcon } from "@/components/ui/icons/Cancel";
import { DiscussionIcon } from "@/components/ui/icons/Discussion";
import { LibraryIcon } from "@/components/ui/icons/Library";
import { ReplyIcon } from "@/components/ui/icons/Reply";
import { SearchIcon } from "@/components/ui/icons/Search";
import { Input } from "@/components/ui/input";
import {
  Box,
  Center,
  Flex,
  HStack,
  VStack,
  styled,
} from "@/styled-system/jsx";
import { vstack } from "@/styled-system/patterns";

import { Props, useSearchScreen } from "./useSearch";
import { useTranslation } from "@/lib/i18n";

export function SearchScreen(props: Props) {
  const t = useTranslation();
  const { ready, form, error, isLoading, data, handlers, filters } =
    useSearchScreen(props);
  const [filtersOpen, setFiltersOpen] = useState(false);

  const { query, page, results } = data;

  const activeFilterCount =
    filters.authorsValue.length +
    filters.categoriesValue.length +
    filters.tagsValue.length;

  return (
    <styled.form
      className={vstack()}
      display="flex"
      w="full"
      gap="3"
      onSubmit={handlers.handleSearch}
      action="/search"
    >
      <HStack
        w="full"
        gap="1"
        pl="3"
        pr="1"
        py="1"
        borderRadius="full"
        borderWidth="thin"
        borderColor="border.default"
        bg="bg.default"
        boxShadow="xs"
        transitionDuration="normal"
        transitionProperty="common"
        _focusWithin={{
          borderColor: "border.accent",
          boxShadow: "sm",
        }}
      >
        <SearchIcon w="5" h="5" flexShrink="0" color="fg.muted" />

        <Input
          flex="1"
          minW="0"
          size="md"
          border="none"
          bg="transparent"
          px="1"
          type="search"
          enterKeyHint="search"
          placeholder={t.search.placeholder}
          _focus={{
            boxShadow: "none" as any,
          }}
          {...form.register("q")}
        />

        {query && (
          <Button
            size="sm"
            variant="ghost"
            borderRadius="full"
            flexShrink="0"
            type="reset"
            onClick={handlers.handleReset}
          >
            <CancelIcon />
          </Button>
        )}

        <Button
          size="sm"
          variant="solid"
          borderRadius="full"
          flexShrink="0"
          type="submit"
          loading={isLoading}
          aria-label={t.search.searchButton}
        >
          <SearchIcon hideFrom="md" />
          <styled.span hideBelow="md">{t.search.searchButton}</styled.span>
        </Button>
      </HStack>

      <HStack w="full" gap="2" alignItems="center">
        <Box flex="1" minW="0">
          <DatagraphKindFilterField
            control={form.control}
            name="kind"
            items={[
              {
                label: t.search.kindThreads,
                description: t.search.kindThreadsDescription,
                icon: <DiscussionIcon />,
                value: DatagraphItemKind.thread,
              },
              {
                label: t.search.kindReplies,
                description: t.search.kindRepliesDescription,
                icon: <ReplyIcon />,
                value: DatagraphItemKind.reply,
              },
              {
                label: t.search.kindLibrary,
                description: t.search.kindLibraryDescription,
                icon: <LibraryIcon />,
                value: DatagraphItemKind.node,
              },
            ]}
          />
        </Box>

        <Button
          hideFrom="md"
          size="sm"
          variant={filtersOpen ? "subtle" : "outline"}
          borderRadius="full"
          flexShrink="0"
          type="button"
          aria-expanded={filtersOpen}
          aria-label={t.search.filters}
          onClick={() => setFiltersOpen((open) => !open)}
        >
          <SlidersHorizontalIcon />
          {activeFilterCount > 0 && (
            <Badge size="sm" borderRadius="full">
              {activeFilterCount}
            </Badge>
          )}
        </Button>
      </HStack>

      <Flex
        w="full"
        gap="2"
        flexDirection={{
          base: "column",
          md: "row",
        }}
        display={{
          base: filtersOpen ? "flex" : "none",
          md: "flex",
        }}
        p={{ base: "3", md: "0" }}
        borderRadius="l3"
        bg={{ base: "bg.subtle", md: "transparent" }}
      >
        <MultiSelectPicker
          value={filters.authorsValue}
          onChange={handlers.handleAuthorsChange}
          onQuery={handlers.handleQueryAuthors}
          queryResults={filters.authorsResults}
          queryError={filters.authorsError}
          inputPlaceholder={t.search.authorsPlaceholder}
          size="sm"
          triggerProps={{
            width: "full",
            minW: "32",
            flexShrink: "1",
          }}
        />

        {filters.showCategories && (
          <MultiSelectPicker
            value={filters.categoriesValue}
            onChange={handlers.handleCategoriesChange}
            onQuery={handlers.handleQueryCategories}
            queryResults={filters.categoriesResults}
            queryError={filters.categoriesError}
            inputPlaceholder={t.search.categoriesPlaceholder}
            size="sm"
            triggerProps={{
              width: "full",
              minW: "32",
              flexShrink: "1",
            }}
          />
        )}

        {filters.showTags && (
          <MultiSelectPicker
            value={filters.tagsValue}
            onChange={handlers.handleTagsChange}
            onQuery={handlers.handleQueryTags}
            queryResults={filters.tagsResults}
            queryError={filters.tagsError}
            inputPlaceholder={t.search.tagsPlaceholder}
            size="sm"
            triggerProps={{
              width: "full",
              minW: "32",
              flexShrink: "1",
            }}
          />
        )}
      </Flex>

      {isLoading || error !== undefined ? (
        <UnreadyBanner error={error} />
      ) : results?.items.length ? (
        <>
          <PaginationControls
            path="/search"
            params={{ q: query }}
            currentPage={page}
            totalPages={results.total_pages}
            pageSize={results.page_size}
          />
          <DatagraphSearchResults result={results} query={query} />
        </>
      ) : (
        <VStack w="full" gap="3" py="16" px="6" textAlign="center">
          <Center
            w="14"
            h="14"
            borderRadius="full"
            bg="bg.subtle"
            borderWidth="thin"
            borderColor="border.subtle"
            color="fg.muted"
          >
            <SearchIcon w="6" h="6" />
          </Center>
          <VStack gap="1" maxW="xs">
            <styled.p fontWeight="semibold" color="fg.default">
              {query
                ? results &&
                  results.total_pages > 0 &&
                  page > results.total_pages
                  ? t.search.pastLastPage
                  : t.search.noResults
                : t.search.emptyPrompt}
            </styled.p>
            {!query && (
              <styled.p textStyle="sm" color="fg.muted">
                {t.search.emptyHint}
              </styled.p>
            )}
          </VStack>
        </VStack>
      )}
    </styled.form>
  );
}

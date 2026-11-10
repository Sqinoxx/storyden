"use client";

import {
  FormEvent,
  PropsWithChildren,
  useEffect,
  useRef,
  useState,
} from "react";

import { useListingList } from "@/api/openapi-client/listings";
import {
  ListingScopeQueryParameter,
  ListingSortQueryParameter,
  Permission,
} from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { ListingCard } from "@/components/marketplace/ListingCard";
import { NativeSelect } from "@/components/marketplace/NativeSelect";
import { PriceRangeSlider } from "@/components/marketplace/PriceRangeSlider";
import {
  conditionOptions,
  priceTypeOptions,
} from "@/components/marketplace/options";
import { useMarketCategories } from "@/components/marketplace/useMarketCategories";
import { PaginationControls } from "@/components/site/PaginationControls/PaginationControls";
import { UnreadyBanner } from "@/components/site/Unready";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { AddIcon } from "@/components/ui/icons/Add";
import { CloseIcon } from "@/components/ui/icons/Close";
import { FilterIcon } from "@/components/ui/icons/Filter";
import { MarketplaceIcon } from "@/components/ui/icons/Marketplace";
import { SearchIcon } from "@/components/ui/icons/Search";
import { Input } from "@/components/ui/input";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { eurosToCents, formatCents } from "@/lib/marketplace/marketplace";
import { Box, Flex, Grid, HStack, LStack, styled } from "@/styled-system/jsx";
import { hasPermission } from "@/utils/permissions";

import { useMarketplaceFilters } from "./useMarketplaceFilters";

const listingGridColumns = {
  base: "repeat(2, minmax(0, 1fr))",
  md: "repeat(3, minmax(0, 1fr))",
  xl: "repeat(4, minmax(0, 1fr))",
};

export function MarketplaceIndexScreen() {
  const t = useTranslation();
  const { filters, params, update } = useMarketplaceFilters();
  const { data, error, mutate } = useListingList(params);
  const marketCategories = useMarketCategories();
  const session = useSession();
  const canManage = hasPermission(session, Permission.MANAGE_LISTINGS);
  const [filtersOpen, setFiltersOpen] = useState(false);

  const lastBounds = useRef({ min: 0, max: 0 });
  if (data) {
    lastBounds.current = {
      min: Math.floor(data.price_min / 100),
      max: Math.ceil(data.price_max / 100),
    };
  }
  const bounds = lastBounds.current;

  const scope = filters.scope ?? ListingScopeQueryParameter.all;

  const [query, setQuery] = useState(filters.q);
  const pushedQuery = useRef(filters.q);

  useEffect(() => {
    if (filters.q !== pushedQuery.current) {
      pushedQuery.current = filters.q;
      setQuery(filters.q);
    }
  }, [filters.q]);

  useEffect(() => {
    if (query === pushedQuery.current) return;
    const timeout = setTimeout(() => {
      pushedQuery.current = query;
      update({ q: query });
    }, 300);
    return () => clearTimeout(timeout);
  }, [query]);

  function handleSearch(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    pushedQuery.current = query;
    update({ q: query });
  }

  function resetFilters() {
    update({
      q: "",
      category: null,
      condition: null,
      price_type: null,
      price_min: null,
      price_max: null,
    });
  }

  const scopes = [
    { value: ListingScopeQueryParameter.all, label: t.marketplace.scopeAll },
    { value: ListingScopeQueryParameter.mine, label: t.marketplace.scopeMine },
    {
      value: ListingScopeQueryParameter.favourites,
      label: t.marketplace.scopeFavourites,
    },
    ...(canManage
      ? [
          {
            value: ListingScopeQueryParameter.manage,
            label: t.marketplace.scopeManage,
          },
        ]
      : []),
  ];

  const categories = [
    { value: null as string | null, label: t.marketplace.all },
    ...marketCategories.options,
  ];

  const priceLabel =
    filters.price_min != null && filters.price_max != null
      ? `${formatCents(eurosToCents(filters.price_min))} – ${formatCents(eurosToCents(filters.price_max))}`
      : filters.price_min != null
        ? `≥ ${formatCents(eurosToCents(filters.price_min))}`
        : filters.price_max != null
          ? `≤ ${formatCents(eurosToCents(filters.price_max))}`
          : null;

  const activeFilters = [
    filters.q && { key: "q", label: `„${filters.q}“`, clear: { q: "" } },
    filters.category && {
      key: "category",
      label: marketCategories.label(filters.category),
      clear: { category: null },
    },
    filters.condition && {
      key: "condition",
      label: t.marketplace.conditions[filters.condition],
      clear: { condition: null },
    },
    filters.price_type && {
      key: "price_type",
      label: t.marketplace.priceTypes[filters.price_type],
      clear: { price_type: null },
    },
    priceLabel && {
      key: "price",
      label: priceLabel,
      clear: { price_min: null, price_max: null },
    },
  ].filter(Boolean) as {
    key: string;
    label: string;
    clear: Parameters<typeof update>[0];
  }[];

  const panelFilterCount = [
    filters.condition,
    filters.price_type,
    filters.price_min,
    filters.price_max,
  ].filter((v) => v != null).length;

  return (
    <LStack gap={{ base: "4", md: "5" }}>
      <Flex
        w="full"
        gap="3"
        direction={{ base: "column", sm: "row" }}
        justify="space-between"
        align={{ base: "stretch", sm: "end" }}
      >
        <LStack gap="1">
          <Heading size="xl">{t.marketplace.title}</Heading>
          <styled.p color="fg.muted">{t.marketplace.description}</styled.p>
        </LStack>
        <LinkButton href="/flohmarkt/new" flexShrink="0">
          <AddIcon />
          {t.marketplace.newListing}
        </LinkButton>
      </Flex>

      <HStack
        w="full"
        gap="1"
        p="1"
        bg="bg.muted"
        borderRadius="lg"
        overflowX="auto"
        style={{ scrollbarWidth: "none" }}
      >
        {scopes.map((s) => (
          <SegmentButton
            key={s.value}
            active={scope === s.value}
            onClick={() =>
              update({
                scope:
                  s.value === ListingScopeQueryParameter.all ? null : s.value,
              })
            }
          >
            {s.label}
          </SegmentButton>
        ))}
      </HStack>

      <styled.form w="full" display="flex" gap="2" onSubmit={handleSearch}>
        <Box position="relative" flex="1" minW="0">
          <Flex
            position="absolute"
            insetY="0"
            left="3"
            align="center"
            pointerEvents="none"
          >
            <SearchIcon w="4" h="4" color="fg.muted" />
          </Flex>
          <Input
            name="q"
            type="search"
            pl="9"
            placeholder={t.marketplace.searchPlaceholder}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            aria-label={t.marketplace.searchPlaceholder}
          />
        </Box>
        <Button type="submit" flexShrink="0">
          {t.nav.search}
        </Button>
      </styled.form>

      <HStack
        w="full"
        gap="2"
        overflowX="auto"
        pb="1"
        style={{ scrollbarWidth: "none" }}
      >
        {categories.map((c) => (
          <Chip
            key={c.value ?? "all"}
            active={(filters.category ?? null) === c.value}
            onClick={() => update({ category: c.value })}
          >
            {c.label}
          </Chip>
        ))}
      </HStack>

      <LStack w="full" gap="3">
        <HStack w="full" gap="2" justify="space-between">
          <styled.span fontSize="sm" color="fg.muted" whiteSpace="nowrap">
            {data
              ? t.marketplace.results.replace("{count}", String(data.results))
              : " "}
          </styled.span>
          <HStack gap="2" minW="0">
            <Button
              type="button"
              size="sm"
              variant="outline"
              hideFrom="md"
              aria-expanded={filtersOpen}
              aria-controls="marketplace-filters"
              onClick={() => setFiltersOpen((o) => !o)}
            >
              <FilterIcon />
              {t.marketplace.filters}
              {panelFilterCount > 0 && (
                <styled.span
                  minW="5"
                  h="5"
                  px="1"
                  borderRadius="full"
                  bg="fg.default"
                  color="bg.default"
                  fontSize="xs"
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                >
                  {panelFilterCount}
                </styled.span>
              )}
            </Button>
            <NativeSelect
              w="auto"
              maxW="48"
              aria-label={t.marketplace.sort}
              value={filters.sort ?? ListingSortQueryParameter.newest}
              onChange={(e) =>
                update({
                  sort:
                    e.target.value === ListingSortQueryParameter.newest
                      ? null
                      : (e.target.value as typeof filters.sort),
                })
              }
              options={[
                {
                  value: ListingSortQueryParameter.newest,
                  label: t.marketplace.sortNewest,
                },
                {
                  value: ListingSortQueryParameter.price_asc,
                  label: t.marketplace.sortPriceAsc,
                },
                {
                  value: ListingSortQueryParameter.price_desc,
                  label: t.marketplace.sortPriceDesc,
                },
              ]}
            />
          </HStack>
        </HStack>

        <Grid
          id="marketplace-filters"
          w="full"
          gap="2"
          p={{ base: "3", md: "0" }}
          bg={{ base: "bg.subtle", md: "transparent" }}
          borderRadius="lg"
          display={{ base: filtersOpen ? "grid" : "none", md: "grid" }}
          gridTemplateColumns={{
            base: "repeat(2, minmax(0, 1fr))",
            md: "repeat(4, minmax(0, 1fr))",
          }}
        >
          <NativeSelect
            aria-label={t.marketplace.form.condition}
            value={filters.condition ?? ""}
            onChange={(e) =>
              update({
                condition: (e.target.value || null) as typeof filters.condition,
              })
            }
            options={[
              { value: "", label: t.marketplace.anyCondition },
              ...conditionOptions(t),
            ]}
          />
          <NativeSelect
            aria-label={t.marketplace.form.priceType}
            value={filters.price_type ?? ""}
            onChange={(e) =>
              update({
                price_type: (e.target.value ||
                  null) as typeof filters.price_type,
              })
            }
            options={[
              { value: "", label: t.marketplace.anyPriceType },
              ...priceTypeOptions(t),
            ]}
          />
          <styled.div gridColumn="span 2" pt={{ base: "1", md: "0" }}>
            <PriceRangeSlider
              key={`${bounds.min}-${bounds.max}-${filters.price_min}-${filters.price_max}`}
              boundsMin={bounds.min}
              boundsMax={bounds.max}
              min={filters.price_min}
              max={filters.price_max}
              onChange={(price_min, price_max) =>
                update({ price_min, price_max })
              }
            />
          </styled.div>
        </Grid>

        {activeFilters.length > 0 && (
          <HStack w="full" gap="1.5" flexWrap="wrap">
            {activeFilters.map((f) => (
              <styled.button
                key={f.key}
                type="button"
                display="inline-flex"
                alignItems="center"
                gap="1"
                pl="2.5"
                pr="1.5"
                py="1"
                fontSize="xs"
                borderRadius="full"
                bg="bg.muted"
                color="fg.default"
                cursor="pointer"
                _hover={{ bg: "bg.emphasized" }}
                onClick={() => update(f.clear)}
              >
                {f.label}
                <CloseIcon w="3" h="3" />
              </styled.button>
            ))}
            <Button
              type="button"
              size="xs"
              variant="ghost"
              onClick={resetFilters}
            >
              {t.marketplace.resetFilters}
            </Button>
          </HStack>
        )}
      </LStack>

      {!data ? (
        error ? (
          <UnreadyBanner error={error} />
        ) : (
          <Grid
            w="full"
            gap={{ base: "2", md: "4" }}
            gridTemplateColumns={listingGridColumns}
          >
            {Array.from({ length: 8 }).map((_, i) => (
              <LStack
                key={i}
                gap="2"
                borderWidth="thin"
                borderStyle="solid"
                borderColor="border.subtle"
                borderRadius="xl"
                overflow="hidden"
                pb="3"
                animation="skeleton-pulse"
              >
                <Box w="full" aspectRatio="[4/3]" bg="bg.muted" />
                <Box mx="3" h="5" w="1/3" bg="bg.muted" borderRadius="md" />
                <Box mx="3" h="4" w="3/4" bg="bg.muted" borderRadius="md" />
              </LStack>
            ))}
          </Grid>
        )
      ) : data.listings.length === 0 ? (
        <LStack
          w="full"
          gap="3"
          py="12"
          px="4"
          alignItems="center"
          textAlign="center"
          borderWidth="thin"
          borderStyle="dashed"
          borderColor="border.default"
          borderRadius="xl"
        >
          <Box p="3" borderRadius="full" bg="bg.muted">
            <MarketplaceIcon w="8" h="8" color="fg.muted" />
          </Box>
          <styled.p fontWeight="medium">
            {activeFilters.length > 0
              ? t.marketplace.emptyFiltered
              : t.marketplace.empty}
          </styled.p>
          {activeFilters.length > 0 ? (
            <Button type="button" variant="outline" onClick={resetFilters}>
              {t.marketplace.resetFilters}
            </Button>
          ) : (
            <styled.p fontSize="sm" color="fg.muted">
              {t.marketplace.emptyHint}
            </styled.p>
          )}
        </LStack>
      ) : (
        <>
          <Grid
            w="full"
            gap={{ base: "2", md: "4" }}
            gridTemplateColumns={listingGridColumns}
          >
            {data.listings.map((l) => (
              <ListingCard key={l.id} listing={l} onChange={() => mutate()} />
            ))}
          </Grid>

          <PaginationControls
            path="/flohmarkt"
            currentPage={data.current_page}
            totalPages={data.total_pages}
            pageSize={data.page_size}
            onClick={(page) => update({ page })}
          />
        </>
      )}
    </LStack>
  );
}

type ToggleProps = PropsWithChildren<{
  active: boolean;
  onClick: () => void;
}>;

function SegmentButton({ active, onClick, children }: ToggleProps) {
  return (
    <styled.button
      type="button"
      flex="1"
      flexShrink="0"
      px="3"
      py="1.5"
      fontSize="sm"
      fontWeight="medium"
      whiteSpace="nowrap"
      borderRadius="md"
      cursor="pointer"
      transitionProperty="common"
      transitionDuration="fast"
      bg={active ? "bg.default" : "transparent"}
      color={active ? "fg.default" : "fg.muted"}
      shadow={active ? "xs" : undefined}
      _hover={{ color: "fg.default" }}
      aria-pressed={active}
      onClick={onClick}
    >
      {children}
    </styled.button>
  );
}

function Chip({ active, onClick, children }: ToggleProps) {
  return (
    <styled.button
      type="button"
      flexShrink="0"
      px="3"
      py="1.5"
      fontSize="sm"
      whiteSpace="nowrap"
      borderRadius="full"
      borderWidth="thin"
      borderStyle="solid"
      cursor="pointer"
      transitionProperty="common"
      transitionDuration="fast"
      borderColor={active ? "fg.default" : "border.default"}
      bg={active ? "fg.default" : "bg.default"}
      color={active ? "bg.default" : "fg.default"}
      _hover={active ? undefined : { bg: "bg.muted" }}
      aria-pressed={active}
      onClick={onClick}
    >
      {children}
    </styled.button>
  );
}

"use client";

import { PlusIcon, SlidersHorizontalIcon, XIcon } from "lucide-react";
import { FormEvent, useEffect, useRef, useState } from "react";

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
import { SearchIcon } from "@/components/ui/icons/Search";
import { Input } from "@/components/ui/input";
import { InputGroup } from "@/components/ui/input-group";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { Grid, HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { hasPermission } from "@/utils/permissions";

import { useMarketplaceFilters } from "./useMarketplaceFilters";

export function MarketplaceIndexScreen() {
  const t = useTranslation();
  const { filters, params, update } = useMarketplaceFilters();
  const { data, error, mutate } = useListingList(params);
  const categories = useMarketCategories();
  const session = useSession();
  const canManage = hasPermission(session, Permission.MANAGE_LISTINGS);
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
  const [filtersOpen, setFiltersOpen] = useState(false);
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

  const activeFilterCount = [
    filters.category,
    filters.condition,
    filters.price_type,
    filters.price_min ?? filters.price_max,
  ].filter((v) => v != null).length;

  function resetFilters() {
    update({
      category: null,
      condition: null,
      price_type: null,
      price_min: null,
      price_max: null,
    });
  }

  return (
    <LStack gap="4">
      <WStack alignItems="start" gap="3">
        <LStack gap="1" minW="0">
          <Heading size="lg">{t.marketplace.title}</Heading>
          <styled.p color="fg.muted" fontSize={{ base: "sm", md: "md" }}>
            {t.marketplace.description}
          </styled.p>
        </LStack>
        <LinkButton href="/flohmarkt/new" size="sm" flexShrink="0">
          <PlusIcon />
          {t.marketplace.newListing}
        </LinkButton>
      </WStack>

      <Grid
        gridTemplateColumns={canManage ? "repeat(4, auto)" : "repeat(3, auto)"}
        gap="1"
        p="1"
        w={{ base: "full", md: "fit" }}
        borderRadius="l2"
        borderWidth="thin"
        borderColor="border.subtle"
        bg="bg.subtle"
      >
        {scopes.map((s) => (
          <Button
            key={s.value}
            type="button"
            size="xs"
            variant={scope === s.value ? "solid" : "ghost"}
            minW="0"
            onClick={() =>
              update({
                scope:
                  s.value === ListingScopeQueryParameter.all ? null : s.value,
              })
            }
          >
            {s.label}
          </Button>
        ))}
      </Grid>

      <styled.form
        w="full"
        display="flex"
        flexDir="column"
        gap="2"
        onSubmit={handleSearch}
      >
        <HStack gap="2" w="full">
          <InputGroup
            flex="1"
            minW="0"
            startElement={<SearchIcon w="4" h="4" color="fg.muted" />}
          >
            <Input
              name="q"
              type="search"
              placeholder={t.marketplace.searchPlaceholder}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              aria-label={t.marketplace.searchPlaceholder}
            />
          </InputGroup>
          <Button
            type="button"
            variant={filtersOpen ? "subtle" : "outline"}
            hideFrom="md"
            flexShrink="0"
            aria-expanded={filtersOpen}
            onClick={() => setFiltersOpen((o) => !o)}
          >
            <SlidersHorizontalIcon />
            {t.marketplace.filters}
            {activeFilterCount > 0 && (
              <styled.span
                display="inline-flex"
                alignItems="center"
                justifyContent="center"
                minW="5"
                h="5"
                px="1"
                borderRadius="full"
                bg="colorPalette.default"
                color="colorPalette.fg"
                fontSize="xs"
                fontWeight="semibold"
              >
                {activeFilterCount}
              </styled.span>
            )}
          </Button>
        </HStack>

        <Grid
          w="full"
          display={{ base: filtersOpen ? "grid" : "none", md: "grid" }}
          gap={{ base: "3", md: "2" }}
          p={{ base: "3", md: "0" }}
          borderRadius="l3"
          borderWidth={{ base: "thin", md: "none" }}
          borderColor="border.subtle"
          bg={{ base: "bg.subtle", md: "transparent" }}
          gridTemplateColumns={{
            base: "repeat(2, 1fr)",
            md: "repeat(3, 1fr)",
            lg: "repeat(6, 1fr)",
          }}
        >
          <NativeSelect
            aria-label={t.marketplace.form.category}
            value={filters.category ?? ""}
            onChange={(e) => update({ category: e.target.value || null })}
            options={[
              { value: "", label: t.marketplace.allCategories },
              ...categories.options,
            ]}
          />
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
          <NativeSelect
            aria-label={t.marketplace.sortNewest}
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
          {activeFilterCount > 0 && (
            <Button
              type="button"
              variant="ghost"
              size="xs"
              gridColumn="span 2"
              hideFrom="md"
              onClick={resetFilters}
            >
              <XIcon />
              {t.marketplace.resetFilters}
            </Button>
          )}
        </Grid>
      </styled.form>

      {!data ? (
        <UnreadyBanner error={error} />
      ) : data.listings.length === 0 ? (
        <styled.p color="fg.muted" py="8" textAlign="center" w="full">
          {t.marketplace.empty}
        </styled.p>
      ) : (
        <>
          <Grid
            w="full"
            gap="3"
            gridTemplateColumns={{
              base: "repeat(2, 1fr)",
              md: "repeat(3, 1fr)",
              xl: "repeat(4, 1fr)",
            }}
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

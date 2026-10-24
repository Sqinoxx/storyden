"use client";

import { FormEvent, useRef } from "react";

import { useListingList } from "@/api/openapi-client/listings";
import {
  ListingScopeQueryParameter,
  ListingSortQueryParameter,
} from "@/api/openapi-schema";
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
import { Input } from "@/components/ui/input";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { Grid, HStack, LStack, WStack, styled } from "@/styled-system/jsx";

import { useMarketplaceFilters } from "./useMarketplaceFilters";

export function MarketplaceIndexScreen() {
  const t = useTranslation();
  const { filters, params, update } = useMarketplaceFilters();
  const { data, error, mutate } = useListingList(params);
  const categories = useMarketCategories();
  const lastBounds = useRef({ min: 0, max: 0 });
  if (data) {
    lastBounds.current = {
      min: Math.floor(data.price_min / 100),
      max: Math.ceil(data.price_max / 100),
    };
  }
  const bounds = lastBounds.current;

  const scope = filters.scope ?? ListingScopeQueryParameter.all;

  function handleSearch(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    update({ q: String(form.get("q") ?? "") });
  }

  const scopes = [
    { value: ListingScopeQueryParameter.all, label: t.marketplace.scopeAll },
    { value: ListingScopeQueryParameter.mine, label: t.marketplace.scopeMine },
    {
      value: ListingScopeQueryParameter.favourites,
      label: t.marketplace.scopeFavourites,
    },
  ];

  return (
    <LStack gap="4">
      <WStack alignItems="start">
        <LStack gap="1">
          <Heading size="lg">{t.marketplace.title}</Heading>
          <styled.p color="fg.muted">{t.marketplace.description}</styled.p>
        </LStack>
        <LinkButton href="/flohmarkt/new" size="sm" flexShrink="0">
          {t.marketplace.newListing}
        </LinkButton>
      </WStack>

      <HStack gap="1" flexWrap="wrap">
        {scopes.map((s) => (
          <Button
            key={s.value}
            type="button"
            size="xs"
            variant={scope === s.value ? "solid" : "subtle"}
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
      </HStack>

      <styled.form
        w="full"
        display="flex"
        flexDir="column"
        gap="2"
        onSubmit={handleSearch}
        key={filters.q}
      >
        <HStack w="full" gap="2">
          <Input
            name="q"
            type="search"
            size="sm"
            placeholder={t.marketplace.searchPlaceholder}
            defaultValue={filters.q}
            aria-label={t.marketplace.searchPlaceholder}
          />
          <Button type="submit" size="sm">
            {t.nav.search}
          </Button>
        </HStack>

        <Grid
          w="full"
          gap="2"
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
          <styled.div gridColumn="span 2">
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

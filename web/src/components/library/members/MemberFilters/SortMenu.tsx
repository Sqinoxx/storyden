"use client";

import { parseAsString, useQueryState } from "nuqs";

import { Button } from "@/components/ui/button";
import { CheckIcon } from "@/components/ui/icons/Check";
import { SortIcon } from "@/components/ui/icons/Sort";
import * as Menu from "@/components/ui/menu";
import { HStack } from "@/styled-system/jsx";
import { useTranslation } from "@/lib/i18n";

const SORT_OPTIONS = [
  { value: "name", label: "Display name (A-Z)" },
  { value: "-name", label: "Display name (Z-A)" },
  { value: "handle", label: "Handle (A-Z)" },
  { value: "-handle", label: "Handle (Z-A)" },
  { value: "created_at", label: "Join date (oldest)" },
  { value: "-created_at", label: "Join date (newest)" },
] as const;

export function SortMenu() {
  const t = useTranslation();
  const [sort, setSort] = useQueryState("sort", parseAsString);

  const sortOptions = [
    { value: "name", label: t.members.sortNameAsc },
    { value: "-name", label: t.members.sortNameDesc },
    { value: "handle", label: t.members.sortHandleAsc },
    { value: "-handle", label: t.members.sortHandleDesc },
    { value: "created_at", label: t.members.sortJoinedAsc },
    { value: "-created_at", label: t.members.sortJoinedDesc },
  ];

  const handleSortChange = async (value: string) => {
    await setSort(value);
  };

  const currentLabel =
    sortOptions.find((opt) => opt.value === sort)?.label || t.members.sortBy;

  return (
    <Menu.Root positioning={{ placement: "bottom-start" }} lazyMount>
      <Menu.Trigger asChild>
        <Button variant="subtle" size="sm" aria-label="Sort options">
          <HStack gap="1">
            <SortIcon />
            {currentLabel}
          </HStack>
        </Button>
      </Menu.Trigger>

      <Menu.Positioner>
        <Menu.Content minW="56">
          <Menu.ItemGroup id="sort-options">
            {sortOptions.map((option) => (
              <Menu.Item
                key={option.value}
                value={option.value}
                onClick={() => handleSortChange(option.value)}
                aria-label={option.label}
              >
                <HStack gap="2" justify="space-between" w="full">
                  <span>{option.label}</span>
                  {sort === option.value && <CheckIcon />}
                </HStack>
              </Menu.Item>
            ))}
          </Menu.ItemGroup>
        </Menu.Content>
      </Menu.Positioner>
    </Menu.Root>
  );
}

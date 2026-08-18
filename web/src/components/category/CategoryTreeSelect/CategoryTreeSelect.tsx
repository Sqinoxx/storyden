"use client";

import { Portal } from "@ark-ui/react";
import { useState } from "react";
import {
  Control,
  Controller,
  FieldPath,
  FieldValues,
  useFormState,
} from "react-hook-form";

import { ErrorTooltip } from "@/components/ui/ErrorTooltip";
import { SelectIcon } from "@/components/ui/icons/Select";
import * as Popover from "@/components/ui/popover";
import { useTranslation } from "@/lib/i18n";
import { findCategoryPath } from "@/lib/category/tree";
import { css } from "@/styled-system/css";
import { HStack, styled } from "@/styled-system/jsx";
import { select } from "@/styled-system/recipes";

import { CategoryTreePanel, EmptyOption } from "./CategoryTreePanel";
import {
  NO_CATEGORY_VALUE,
  SelectableMode,
  useCategoryTreeSelect,
} from "./useCategoryTreeSelect";

export type CategoryTreeSelectProps<T extends FieldValues> = {
  control: Control<T>;
  name: FieldPath<T>;
  rootId?: string;
  selectable?: SelectableMode;
  emptyOption?: EmptyOption | "auto";
};

const triggerStyles = css({
  gap: "1",
  h: "8",
  minW: "32",
  maxW: "full",
  px: "3",
  textStyle: "sm",
});

export function CategoryTreeSelect<T extends FieldValues>({
  control,
  name,
  rootId,
  selectable,
  emptyOption = "auto",
}: CategoryTreeSelectProps<T>) {
  const t = useTranslation();
  const [isOpen, setIsOpen] = useState(false);

  const { ready, error, tree, canPostAnywhere, isSelectable } =
    useCategoryTreeSelect({ rootId, selectable });

  const { errors } = useFormState({ control, name });
  const fieldError = errors[name]?.message as string | undefined;

  const styles = select({ size: "sm" });

  const resolvedEmptyOption =
    emptyOption === "auto"
      ? canPostAnywhere
        ? { label: t.category.noCategory, value: NO_CATEGORY_VALUE }
        : undefined
      : emptyOption;

  return (
    <HStack gap="2" alignItems="center" minW="0">
      <Controller<T>
        control={control}
        name={name}
        render={({ field }) => {
          const path = findCategoryPath(tree, field.value);

          const label = (() => {
            if (path.length > 0) {
              return path.map((node) => node.name).join(" / ");
            }

            if (resolvedEmptyOption && field.value === resolvedEmptyOption.value) {
              return resolvedEmptyOption.label;
            }

            return undefined;
          })();

          function handleSelect(value: string) {
            field.onChange(value);
            setIsOpen(false);
          }

          return (
            <Popover.Root
              lazyMount
              unmountOnExit
              open={isOpen}
              onOpenChange={(details) => setIsOpen(details.open)}
              positioning={{ placement: "bottom-start", gutter: 4 }}
            >
              <Popover.Trigger asChild>
                <styled.button
                  type="button"
                  disabled={!ready}
                  className={`${styles.trigger} ${triggerStyles}`}
                >
                  <styled.span
                    truncate
                    color={label ? "fg.default" : "fg.subtle"}
                  >
                    {ready
                      ? (label ?? t.category.selectCategory)
                      : t.category.loadingCategories}
                  </styled.span>
                  <SelectIcon />
                </styled.button>
              </Popover.Trigger>

              <Portal>
                <Popover.Positioner>
                  <Popover.Content
                    w="[min(20rem, calc(100vw - 2rem))]"
                    maxW="[calc(100vw - 2rem)]"
                    overflowY="auto"
                  >
                    <CategoryTreePanel
                      tree={tree}
                      value={field.value}
                      isSelectable={isSelectable}
                      onSelect={handleSelect}
                      emptyOption={resolvedEmptyOption}
                    />
                  </Popover.Content>
                </Popover.Positioner>
              </Portal>
            </Popover.Root>
          );
        }}
      />

      <ErrorTooltip error={fieldError ?? error} />
    </HStack>
  );
}

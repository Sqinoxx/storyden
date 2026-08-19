import { Portal } from "@ark-ui/react";
import { PlusIcon, TagIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import * as Menu from "@/components/ui/menu";
import { Box, HStack, WStack } from "@/styled-system/jsx";
import {
  ButtonVariantProps,
  InputVariantProps,
  MenuVariantProps,
  input,
} from "@/styled-system/recipes";
import { deriveColour } from "@/utils/colour";

import { CancelAction } from "../site/Action/Cancel";
import { Unready } from "../site/Unready";

import { Badge, badgeColourPalette, badgeColours } from "./badge";
import { ButtonProps } from "./button";
import { Input } from "./input";
import { Text } from "./text";

export type MultiSelectPickerItem = {
  label: string;
  value: string;
  colour?: string;
};

type Props = {
  value: MultiSelectPickerItem[];
  initialValue?: MultiSelectPickerItem[];
  onChange: (item: MultiSelectPickerItem[]) => Promise<void>;
  onQuery: (query: string) => void;
  queryResults?: MultiSelectPickerItem[];
  allowNewValues?: boolean;
  inputPlaceholder?: string;
  autoColour?: boolean;
  queryError?: string | null;

  // styling
  size?: ButtonVariantProps["size"] & MenuVariantProps["size"];
  triggerProps?: ButtonProps;
  menuVariantProps?: MenuVariantProps;
  inputVariantProps?: InputVariantProps;
};

export function MultiSelectPicker({
  value,
  onChange,
  onQuery,
  queryResults,
  allowNewValues,
  inputPlaceholder,
  autoColour,
  queryError,

  size,
  triggerProps,
  menuVariantProps,
  inputVariantProps,
}: Props) {
  const [queryInput, setQueryInput] = useState("");
  const [hiddenCount, setHiddenCount] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const badgeRefs = useRef<(HTMLDivElement | null)[]>([]);

  useEffect(() => {
    const calculateHiddenItems = () => {
      if (!containerRef.current || value.length === 0) {
        setHiddenCount(0);
        return;
      }

      const containerRect = containerRef.current.getBoundingClientRect();
      const containerRight = containerRect.right;
      let hidden = 0;

      badgeRefs.current.forEach((badge) => {
        if (!badge) return;

        const badgeRect = badge.getBoundingClientRect();
        const badgeRight = badgeRect.right;
        const badgeLeft = badgeRect.left;
        const badgeWidth = badgeRect.width;

        if (badgeLeft >= containerRight) {
          hidden++;
        } else if (badgeRight > containerRight) {
          const visibleWidth = containerRight - badgeLeft;
          const visiblePercentage = (visibleWidth / badgeWidth) * 100;

          if (visiblePercentage < 50) {
            hidden++;
          }
        }
      });

      setHiddenCount(hidden);
    };

    calculateHiddenItems();

    const resizeObserver = new ResizeObserver(calculateHiddenItems);
    if (containerRef.current) {
      resizeObserver.observe(containerRef.current);
    }

    return () => {
      resizeObserver.disconnect();
    };
  }, [value]);

  function handleQuery(event: React.ChangeEvent<HTMLInputElement>) {
    const queryValue = event.target.value;
    setQueryInput(queryValue);
    onQuery(queryValue);
  }

  const handleRemoveItem =
    (item: MultiSelectPickerItem) => async (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      const newValue = value.filter((v) => v.value !== item.value);
      await onChange(newValue);
    };

  const handleAddResult = (item: MultiSelectPickerItem) => async () => {
    if (value.some((v) => v.value === item.value)) {
      return;
    }

    const itemWithColour =
      autoColour && !item.colour
        ? { ...item, colour: deriveColour(item.value) }
        : item;

    await onChange([...value, itemWithColour]);
  };

  const handleAddNewValue = async () => {
    if (!allowNewValues || !queryInput.trim()) return;

    const newItem: MultiSelectPickerItem = {
      label: queryInput,
      value: queryInput,
      colour: autoColour ? deriveColour(queryInput) : undefined,
    };

    await onChange([...value, newItem]);
    setQueryInput("");
  };

  const handleKeyDown = async (
    event: React.KeyboardEvent<HTMLInputElement>,
  ) => {
    if (event.key === "Enter" && allowNewValues && queryInput.trim()) {
      event.preventDefault();
      await handleAddNewValue();
    }
  };

  const filteredQueryResults = queryResults?.filter(
    (result) => !value.some((v) => v.value === result.value),
  );

  const showCreateNew =
    allowNewValues &&
    queryInput.trim() &&
    !value.some((v) => v.value === queryInput) &&
    (!filteredQueryResults ||
      !filteredQueryResults.some((r) => r.value === queryInput));

  return (
    <Menu.Root
      size={size}
      lazyMount
      positioning={{
        placement: "bottom-start",
        strategy: "fixed",
      }}
      {...menuVariantProps}
    >
      <Menu.Trigger
        w="full"
        flexShrink="1"
        justifyContent="space-between"
        cursor="pointer"
        aria-label={`Select items${value.length > 0 ? `, ${value.length} selected` : ""}`}
        {...triggerProps}
      >
        <HStack
          ref={containerRef}
          gap="1"
          w="full"
          className={input({
            size,
          })}
          overflowX="clip"
          overflowY="hidden"
          position="relative"
        >
          {value.length > 0 ? (
            <>
              {value.map((item, index) => {
                const colour = badgeColours(
                  item.colour ? item.colour : deriveColour(item.value),
                );

                const colourStyles = colour
                  ? badgeColourPalette(colour)
                  : undefined;

                return (
                  <Badge
                    key={item.value}
                    ref={(el) => {
                      badgeRefs.current[index] = el;
                    }}
                    style={colourStyles}
                    bgColor={colourStyles ? "colorPalette.bg" : undefined}
                    borderColor={
                      colourStyles ? "colorPalette.border" : undefined
                    }
                    color={colourStyles ? "colorPalette.fg" : undefined}
                  >
                    {item.label}
                  </Badge>
                );
              })}
              {hiddenCount > 0 && (
                <>
                  <HStack
                    position="absolute"
                    right="-0.5"
                    bg="bg.muted"
                    color="fg.default"
                    backdropBlur="frosted"
                    backdropFilter="auto"
                    mask="linear-gradient(to right, rgb(from {colors.bg.subtle} r g b / 0) 0%, rgb(from {colors.bg.subtle} r g b / 0.8) 100%)"
                    fontWeight="semibold"
                    pointerEvents="none"
                    height="full"
                    alignItems="center"
                    px="2"
                  >
                    <Box>+{hiddenCount}</Box>
                  </HStack>
                  <HStack
                    position="absolute"
                    right="-0.5"
                    color="fg.default"
                    bg="overflow-fade"
                    fontWeight="semibold"
                    pointerEvents="none"
                    height="full"
                    alignItems="center"
                    px="2"
                  >
                    <Box>+{hiddenCount}</Box>
                  </HStack>
                </>
              )}
            </>
          ) : (
            <Text size="sm" color="fg.muted" textWrap="nowrap">
              {inputPlaceholder || "Select items..."}
            </Text>
          )}
        </HStack>
      </Menu.Trigger>

      <Portal>
        <Menu.Positioner zIndex="popover">
          <Menu.Content zIndex="popover" style={{ minWidth: "240px", maxHeight: "320px", overflowY: "auto" }}>
            <Box position="sticky" top="0" zIndex="[1]" bg="bg.default" p="2" pb="1" borderBottomWidth="thin" borderColor="border.subtle">
              <Input
                size={size}
                value={queryInput}
                placeholder="Search..."
                aria-label="Search for items"
                {...inputVariantProps}
                onChange={handleQuery}
                onKeyDown={handleKeyDown}
              />
            </Box>

            {value.length > 0 && (
              <Menu.ItemGroup pt="1">
                <Menu.ItemGroupLabel>Selected</Menu.ItemGroupLabel>
                {value.map((item) => {
                  return (
                    <Menu.Item
                      key={item.value}
                      value={item.value}
                      closeOnSelect={false}
                      asChild
                    >
                      <WStack alignItems="center" cursor="default">
                        <span>{item.label}</span>

                        <CancelAction
                          onClick={handleRemoveItem(item)}
                          aria-label={`Remove ${item.label}`}
                        />
                      </WStack>
                    </Menu.Item>
                  );
                })}
              </Menu.ItemGroup>
            )}

            {queryError ? (
              <Menu.ItemGroup p="2">
                <Unready error={queryError} />
              </Menu.ItemGroup>
            ) : (
              <>
                {filteredQueryResults && filteredQueryResults.length > 0 && (
                  <Menu.ItemGroup
                    borderTopWidth={value.length > 0 ? "thin" : undefined}
                    borderColor={value.length > 0 ? "border.subtle" : undefined}
                    mt={value.length > 0 ? "2" : undefined}
                    pt={value.length > 0 ? "1" : undefined}
                  >
                    <Menu.ItemGroupLabel>
                      <HStack gap="1.5" alignItems="center">
                        <TagIcon size={12} />
                        <span>
                          {queryInput.trim() ? "Suchergebnisse" : "Vorhandene Tags"}
                        </span>
                      </HStack>
                    </Menu.ItemGroupLabel>
                    {filteredQueryResults.map((item) => {
                      const colour = badgeColours(
                        item.colour ? item.colour : deriveColour(item.value),
                      );
                      const colourStyles = colour
                        ? badgeColourPalette(colour)
                        : undefined;

                      return (
                        <Menu.Item
                          key={item.value}
                          value={item.value}
                          closeOnSelect={false}
                          onSelect={handleAddResult(item)}
                          cursor="pointer"
                        >
                          <Badge
                            size="sm"
                            style={colourStyles}
                            bgColor={colourStyles ? "colorPalette.bg" : undefined}
                            borderColor={
                              colourStyles ? "colorPalette.border" : undefined
                            }
                            color={colourStyles ? "colorPalette.fg" : undefined}
                          >
                            {item.label}
                          </Badge>
                        </Menu.Item>
                      );
                    })}
                  </Menu.ItemGroup>
                )}

                {showCreateNew && (
                  <Menu.ItemGroup mt="1">
                    <Menu.ItemGroupLabel>
                      <HStack gap="1.5" alignItems="center" color="accent.default">
                        <PlusIcon size={12} />
                        <span>Neuen Tag erstellen</span>
                      </HStack>
                    </Menu.ItemGroupLabel>
                    <Menu.Item
                      value={`new-${queryInput}`}
                      closeOnSelect={false}
                      onSelect={handleAddNewValue}
                      cursor="pointer"
                      style={{
                        borderRadius: "var(--radii-l2)",
                        border: "1px dashed var(--colors-border-default)",
                      }}
                      _hover={{
                        borderColor: "var(--colors-accent-default, #3b82f6)",
                        bg: "var(--colors-bg-muted)",
                      }}
                    >
                      <HStack gap="2" alignItems="center" width="full">
                        <Box
                          p="1"
                          borderRadius="full"
                          bg="accent.default"
                          color="white"
                          display="flex"
                          alignItems="center"
                          justifyContent="center"
                        >
                          <PlusIcon size={12} />
                        </Box>
                        <Text size="sm" fontWeight="medium" color="fg.default">
                          Tag <strong>"{queryInput.trim()}"</strong> erstellen
                        </Text>
                      </HStack>
                    </Menu.Item>
                  </Menu.ItemGroup>
                )}

                {queryInput &&
                  !queryError &&
                  !filteredQueryResults?.length &&
                  !showCreateNew && (
                    <Menu.ItemGroup p="2">
                      <Text size="sm" color="fg.subtle">
                        No results found
                      </Text>
                    </Menu.ItemGroup>
                  )}
              </>
            )}
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  );
}

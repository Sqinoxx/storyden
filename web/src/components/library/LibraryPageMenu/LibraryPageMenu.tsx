"use client";

import { MenuOpenChangeDetails, Portal } from "@ark-ui/react";
import { format } from "date-fns/format";

import { MoreAction } from "@/components/site/Action/More";

import { ReportNodeMenuItem } from "@/components/report/ReportNodeMenuItem";
import { CancelAction } from "@/components/site/Action/Cancel";
import { ButtonProps } from "@/components/ui/button";
import * as Menu from "@/components/ui/menu";
import { HStack, styled } from "@/styled-system/jsx";
import { menuItemColorPalette } from "@/styled-system/patterns";
import { useTranslation } from "@/lib/i18n";

import { Props, useLibraryPageMenu } from "./useLibraryPageMenu";

// For some crazy reason, Ark doesn't export these, so we gotta re-define.
interface EventDetails<T> {
  originalEvent: T;
  contextmenu: boolean;
  focusable: boolean;
  target: EventTarget;
}

type PointerDownOutsideEvent = CustomEvent<EventDetails<PointerEvent>>;
type FocusOutsideEvent = CustomEvent<EventDetails<FocusEvent>>;
type InteractOutsideEvent = PointerDownOutsideEvent | FocusOutsideEvent;

type LibraryPageMenuProps = Props &
  ButtonProps & {
    onOpenChange?: (details: MenuOpenChangeDetails) => void;
    onInteractOutside?: (event: InteractOutsideEvent) => void;
  };

export function LibraryPageMenu({
  node,
  parentID,
  open,
  onClose,
  onOpenChange,
  onInteractOutside,
  children,
  ...props
}: LibraryPageMenuProps) {
  const {
    availableOperations,
    deleteEnabled,
    isChildrenHidden,
    isConfirmingDelete,
    isManager,
    handlers,
  } = useLibraryPageMenu({
    node,
    onClose,
    parentID,
  });

  function handleOpenChange(d: MenuOpenChangeDetails) {
    onOpenChange?.(d);
    if (!d.open) {
      onClose?.();
    }
  }

  const t = useTranslation();

  function getOpLabel(label: string) {
    switch (label) {
      case "Publish to library":
        return t.library.publishToLibrary;
      case "Submit for review":
        return t.library.submitForReview;
      case "Publish to profile":
        return t.library.publishToProfile;
      case "Revert to draft":
        return t.library.revertToDraft;
      case "Unpublish":
        return t.library.unpublish;
      case "Reject":
        return t.library.reject;
      default:
        return label;
    }
  }

  const statusText =
    node.visibility === "draft"
      ? t.library.draft
      : node.visibility === "review"
        ? t.library.inReview
        : "";

  return (
    <Menu.Root
      open={open}
      lazyMount
      positioning={{ placement: "right-start", gutter: -2 }}
      onSelect={handlers.handleSelect}
      onOpenChange={handleOpenChange}
      onInteractOutside={onInteractOutside}
    >
      <Menu.Trigger asChild>
        {children ?? <MoreAction variant="subtle" size="xs" {...props} />}
      </Menu.Trigger>

      <Portal>
        <Menu.Positioner>
          <Menu.Content minW="36">
            <Menu.ItemGroup>
              <Menu.ItemGroupLabel
                display="flex"
                flexDir="column"
                userSelect="none"
              >
                <styled.span>
                  {t.library.createdBy.replace("{name}", node.owner.name)} {statusText}
                </styled.span>

                <styled.time fontWeight="normal">
                  {format(new Date(node.createdAt), "yyyy-mm-dd")}
                </styled.time>
              </Menu.ItemGroupLabel>

              <Menu.Separator />

              {availableOperations.map((op) => (
                <Menu.Item
                  key={op.targetVisibility}
                  value={op.targetVisibility}
                >
                  {getOpLabel(op.label)}
                </Menu.Item>
              ))}

              <ReportNodeMenuItem node={node} />

              {isManager && (
                <Menu.Item value="toggle-hide-in-tree">
                  {isChildrenHidden
                    ? t.library.showChildrenInTree
                    : t.library.hideChildrenInTree}
                </Menu.Item>
              )}

              {deleteEnabled &&
                (isConfirmingDelete ? (
                  <HStack gap="0">
                    <Menu.Item
                      className={menuItemColorPalette()}
                      colorPalette="red"
                      value="delete"
                      w="full"
                      closeOnSelect={false}
                    >
                      {t.actions.deleteConfirm}
                    </Menu.Item>

                    <Menu.Item
                      value="delete-cancel"
                      closeOnSelect={false}
                      asChild
                    >
                      <CancelAction
                        borderRadius="md"
                        onClick={handlers.handleCancelDelete}
                      />
                    </Menu.Item>
                  </HStack>
                ) : (
                  <Menu.Item
                    className={menuItemColorPalette({ colorPalette: "red" })}
                    colorPalette="red"
                    value="delete"
                    closeOnSelect={false}
                  >
                    {t.actions.delete}
                  </Menu.Item>
                ))}
            </Menu.ItemGroup>
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  );
}

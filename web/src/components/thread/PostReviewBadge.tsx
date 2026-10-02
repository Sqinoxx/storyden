"use client";

import { Portal } from "@ark-ui/react";

import { CancelAction } from "@/components/site/Action/Cancel";
import { Badge } from "@/components/ui/badge";
import { CheckIcon } from "@/components/ui/icons/Check";
import { DeleteIcon } from "@/components/ui/icons/Delete";
import { EditIcon } from "@/components/ui/icons/Edit";
import { WarningIcon } from "@/components/ui/icons/Warning";
import * as Menu from "@/components/ui/menu";
import * as Tooltip from "@/components/ui/tooltip";
import { HStack } from "@/styled-system/jsx";
import { menuItemColorPalette } from "@/styled-system/patterns";
import { useTranslation } from "@/lib/i18n";

type Props = {
  isModerator: boolean;
  postId: string;
  onAccept: (postId: string) => void;
  onEditAndAccept: (postId: string) => void;
  onDelete: (postId: string) => void;
  isConfirmingDelete: boolean;
  onCancelDelete: () => void;
};

export function PostReviewBadge({
  isModerator,
  postId,
  onAccept,
  onEditAndAccept,
  onDelete,
  isConfirmingDelete,
  onCancelDelete,
}: Props) {
  const t = useTranslation();

  if (!isModerator) {
    return (
      <Tooltip.Root
        openDelay={0}
        positioning={{
          slide: true,
          shift: 16,
        }}
      >
        <Tooltip.Trigger asChild>
          <Badge
            variant="subtle"
            cursor="pointer"
            aria-label={t.thread.postInReview}
          >
            <WarningIcon />
            {t.thread.inReview}
          </Badge>
        </Tooltip.Trigger>
        <Portal>
          <Tooltip.Positioner>
            <Tooltip.Arrow>
              <Tooltip.ArrowTip />
            </Tooltip.Arrow>

            <Tooltip.Content p="2" borderRadius="2xl" maxW="xs">
              {t.thread.inReviewDescription}
            </Tooltip.Content>
          </Tooltip.Positioner>
        </Portal>
      </Tooltip.Root>
    );
  }

  return (
    <Menu.Root
      positioning={{
        placement: "bottom-start",
      }}
      lazyMount
    >
      <Menu.Trigger asChild>
        <Badge
          variant="subtle"
          cursor="pointer"
          _hover={{
            borderColor: "colorPalette.6",
          }}
          aria-label={t.thread.reviewActions}
        >
          <WarningIcon />
          {t.thread.inReview}
        </Badge>
      </Menu.Trigger>

      <Menu.Positioner>
        <Menu.Content minW="48">
          <Menu.ItemGroup id="review-actions">
            <Menu.Item
              value="accept"
              onClick={() => onAccept(postId)}
              aria-label={t.actions.approve}
            >
              <HStack gap="1">
                <CheckIcon /> {t.actions.approve}
              </HStack>
            </Menu.Item>

            <Menu.Item
              value="edit-and-accept"
              onClick={() => onEditAndAccept(postId)}
              aria-label={t.actions.editAndAccept}
            >
              <HStack gap="1">
                <EditIcon /> {t.actions.editAndAccept}
              </HStack>
            </Menu.Item>

            <Menu.Separator />

            {isConfirmingDelete ? (
              <HStack gap="0">
                <Menu.Item
                  className={menuItemColorPalette({ colorPalette: "red" })}
                  value="confirm-delete"
                  w="full"
                  closeOnSelect={false}
                  onClick={() => onDelete(postId)}
                  aria-label={t.actions.deleteConfirm}
                >
                  {t.actions.deleteConfirm}
                </Menu.Item>

                <Menu.Item
                  value="cancel-delete"
                  closeOnSelect={false}
                  asChild
                  aria-label={t.common.cancel}
                >
                  <CancelAction borderRadius="md" onClick={onCancelDelete} />
                </Menu.Item>
              </HStack>
            ) : (
              <Menu.Item
                className={menuItemColorPalette({ colorPalette: "red" })}
                value="delete"
                closeOnSelect={false}
                onClick={() => onDelete(postId)}
                aria-label={t.actions.delete}
              >
                <HStack gap="1">
                  <DeleteIcon /> {t.actions.delete}
                </HStack>
              </Menu.Item>
            )}
          </Menu.ItemGroup>
        </Menu.Content>
      </Menu.Positioner>
    </Menu.Root>
  );
}

"use client";

import { CancelAction } from "@/components/site/Action/Cancel";
import { DeleteIcon } from "@/components/ui/icons/Delete";
import * as Menu from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";
import { HStack } from "@/styled-system/jsx";
import { menuItemColorPalette } from "@/styled-system/patterns";

import { useConfirmation } from "./useConfirmation";

export type Props = {
  onDelete: () => Promise<void>;
};

export function DeleteWithConfirmationMenuItem(props: Props) {
  const t = useTranslation();
  const { isConfirming, handleConfirmAction, handleCancelAction } =
    useConfirmation(props.onDelete);

  if (isConfirming) {
    return (
      <HStack gap="0">
        <Menu.Item
          className={menuItemColorPalette({ colorPalette: "red" })}
          value="delete"
          w="full"
          closeOnSelect={false}
          onClick={props.onDelete}
        >
          {t.actions.deleteConfirm}
        </Menu.Item>

        <Menu.Item value="delete-cancel" closeOnSelect={false} asChild>
          <CancelAction borderRadius="md" onClick={handleCancelAction} />
        </Menu.Item>
      </HStack>
    );
  }

  return (
    <Menu.Item
      className={menuItemColorPalette({ colorPalette: "red" })}
      value="delete"
      closeOnSelect={false}
      onClick={handleConfirmAction}
    >
      <HStack gap="1">
        <DeleteIcon /> {t.actions.delete}
      </HStack>
    </Menu.Item>
  );
}

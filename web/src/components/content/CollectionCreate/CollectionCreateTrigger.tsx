import { PropsWithChildren } from "react";

import { useDisclosure } from "@/utils/useDisclosure";

import { Button } from "@/components/ui/button";
import { CreateFolderIcon } from "@/components/ui/icons/CreateFolder";
import { ButtonVariantProps } from "@/styled-system/recipes";

import { CollectionCreateModal } from "./CollectionCreateModal";
import { Props } from "./useCollectionCreate";
import { useTranslation } from "@/lib/i18n";

export function CollectionCreateTrigger(
  props: PropsWithChildren<
    ButtonVariantProps &
      Props & {
        label?: string;
      }
  >,
) {
  const t = useTranslation();
  const { onOpen, isOpen, onClose } = useDisclosure();
  return (
    <>
      {props.children ?? (
        <Button
          flexShrink="0"
          minW="0"
          variant="subtle"
          justifyContent="start"
          size="sm"
          {...props}
          onClick={onOpen}
        >
          <CreateFolderIcon /> {props.label ?? t.collections.collection}
        </Button>
      )}
      <CollectionCreateModal isOpen={isOpen} onClose={onClose} {...props} />
    </>
  );
}

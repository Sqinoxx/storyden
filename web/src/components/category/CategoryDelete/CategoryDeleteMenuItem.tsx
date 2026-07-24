"use client";

import { Category } from "@/api/openapi-schema";
import { useDisclosure } from "@/utils/useDisclosure";

import { Item } from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";

import { CategoryDeleteModal } from "./CategoryDeleteModal";

export function CategoryDeleteMenuItem(props: Category) {
  const { onOpen, isOpen, onClose } = useDisclosure();
  const t = useTranslation();

  return (
    <>
      <Item value="delete" onClick={onOpen}>
        {t.actions.delete}
      </Item>
      <CategoryDeleteModal
        onClose={onClose}
        isOpen={isOpen}
        categorySlug={props.slug}
        categoryName={props.name}
      />
    </>
  );
}

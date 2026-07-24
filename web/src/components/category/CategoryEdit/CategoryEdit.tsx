"use client";

import { Category } from "@/api/openapi-schema";
import { useDisclosure } from "@/utils/useDisclosure";

import { Item } from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";

import { CategoryEditModal } from "./CategoryEditModal";

export function CategoryEditMenuItem(props: Category) {
  const disclosure = useDisclosure();
  const t = useTranslation();

  return (
    <>
      <Item value="edit" onClick={disclosure.onOpen}>
        {t.actions.edit}
      </Item>
      <CategoryEditModal {...disclosure} category={props} />
    </>
  );
}

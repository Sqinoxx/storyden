"use client";

import { Category } from "@/api/openapi-schema";
import { useDisclosure } from "@/utils/useDisclosure";

import { Item } from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";

import { CategoryCreateModal } from "./CategoryCreateModal";

type Props = {
  parentCategory?: Category;
};

export function CategoryCreateMenuItem({ parentCategory }: Props) {
  const useDisclosureProps = useDisclosure();
  const t = useTranslation();

  return (
    <>
      <Item value="create-subcategory" onClick={useDisclosureProps.onOpen}>
        {t.category.createSubcategory}
      </Item>
      <CategoryCreateModal
        defaultParent={parentCategory?.id}
        {...useDisclosureProps}
      />
    </>
  );
}

import { createListCollection } from "@ark-ui/react";

import { useCategoryList } from "@/api/openapi-client/categories";

import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { SelectField } from "@/components/ui/form/SelectField";
import { HStack, LStack } from "@/styled-system/jsx";

import { CategoryDeleteProps, useCategoryDelete } from "./useCategoryDelete";
import { useTranslation } from "@/lib/i18n";

export type { CategoryDeleteProps };

export function CategoryDeleteScreen(props: CategoryDeleteProps) {
  const t = useTranslation();
  const { form, handleDelete } = useCategoryDelete(props);
  const { data: categoryListResult } = useCategoryList();
  const categories = categoryListResult?.categories || [];

  // Filter out the category being deleted from the move options
  const availableCategories = categories.filter(
    (category) => category.slug !== props.categorySlug,
  );

  const categoryCollection = createListCollection({
    items: availableCategories.map((category) => ({
      label: category.name,
      value: category.id,
    })),
  });

  return (
    <form onSubmit={handleDelete}>
      <LStack gap="6">
        <LStack gap="4">
          <p>
            <strong>{t.category.deleteWarningLabel}</strong>{" "}
            {t.category.deleteWarning.replace("{name}", props.categoryName)}
          </p>
          <p>
            {t.category.deleteMoveHint}
          </p>
        </LStack>

        <FormControl>
          <FormLabel>{t.category.movePostsTo}</FormLabel>
          <SelectField
            name="move_to"
            control={form.control}
            collection={categoryCollection}
            placeholder={t.category.selectCategoryPlaceholder}
          />
          {form.formState.errors.move_to && (
            <FormErrorText>
              {form.formState.errors.move_to.message}
            </FormErrorText>
          )}
        </FormControl>

        <HStack w="full" justify="end" gap="4">
          <Button type="button" variant="ghost" onClick={props.onClose}>
            {t.common.cancel}
          </Button>
          <Button
            type="submit"
            colorPalette="red"
            loading={form.formState.isSubmitting}
          >
            {t.category.deleteTitle}
          </Button>
        </HStack>
      </LStack>
    </form>
  );
}

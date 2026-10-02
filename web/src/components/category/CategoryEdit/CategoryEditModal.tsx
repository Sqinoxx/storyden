import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";

import { AssetUploadEditor } from "@/components/asset/AssetUploadEditor/AssetUploadEditor";
import { ColourPickerField } from "@/components/ui/ColourPickerField";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormFeedback } from "@/components/ui/form/FormFeedback";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Input, InputPrefix } from "@/components/ui/input";
import { WEB_ADDRESS } from "@/config";
import {
  CATEGORY_COVER_HEIGHT,
  CATEGORY_COVER_WIDTH,
} from "@/lib/category/cover";
import { HStack, VStack, styled } from "@/styled-system/jsx";

import { Props, useCategoryEdit } from "./useCategoryEdit";
import { useTranslation } from "@/lib/i18n";

export function CategoryEditModal(props: Props) {
  const t = useTranslation();
  const { form, handlers } = useCategoryEdit(props);

  const hostname = new URL(WEB_ADDRESS).host;

  return (
    <ModalDrawer
      isOpen={props.isOpen}
      onClose={props.onClose}
      onOpenChange={props.onOpenChange}
      title={t.category.editTitle}
    >
      <styled.form
        display="flex"
        flexDir="column"
        justifyContent="space-between"
        alignItems="start"
        height="full"
        onSubmit={handlers.handleSubmit}
        gap="2"
      >
        <VStack w="full">
          <FormControl>
            <FormLabel>{t.category.coverImage}</FormLabel>
            <AssetUploadEditor
              width={CATEGORY_COVER_WIDTH}
              height={CATEGORY_COVER_HEIGHT}
              value={form.watch("cover_image") || undefined}
              onUpload={handlers.handleImageUpload}
              onRemove={handlers.handleImageRemove}
            />
            <FormFeedback error={form.formState.errors["cover_image"]?.message}>
              {t.category.coverImageHelper}
            </FormFeedback>
          </FormControl>

          <HStack w="full" alignItems="start">
            <FormControl>
              <FormLabel>{t.category.name}</FormLabel>
              <Input {...form.register("name")} type="text" />
              <FormFeedback error={form.formState.errors["name"]?.message}>
                {t.category.nameHelper}
              </FormFeedback>
            </FormControl>

            <FormControl>
              <FormLabel>{t.category.urlSlug}</FormLabel>
              <HStack gap="0" alignItems="stretch" flex="1">
                <InputPrefix
                  display={{
                    base: "none",
                    sm: "flex",
                  }}
                >
                  {hostname}/d/
                </InputPrefix>
                <Input
                  {...form.register("slug")}
                  type="text"
                  flex="1"
                  borderTopLeftRadius={{
                    base: "sm",
                    sm: "none",
                  }}
                  borderBottomLeftRadius={{
                    base: "sm",
                    sm: "none",
                  }}
                />
              </HStack>
              <FormFeedback error={form.formState.errors["slug"]?.message}>
                {t.category.slugHelper}
              </FormFeedback>
            </FormControl>
          </HStack>

          <FormControl>
            <FormLabel>{t.category.description}</FormLabel>
            <Input {...form.register("description")} type="text" />
            <FormFeedback error={form.formState.errors["description"]?.message}>
              {t.category.descriptionHelper}
            </FormFeedback>
          </FormControl>

          <FormControl>
            <FormLabel>{t.category.colour}</FormLabel>
            <ColourPickerField control={form.control} name="colour" />
            <FormFeedback error={form.formState.errors["colour"]?.message}>
              {t.category.colourHelper}
            </FormFeedback>
          </FormControl>
        </VStack>

        <HStack w="full" alignItems="center" justify="end" pb="3" gap="4">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={handlers.handleCancel}
          >
            {t.common.cancel}
          </Button>
          <Button type="submit" size="sm">
            {t.common.save}
          </Button>
        </HStack>
      </styled.form>
    </ModalDrawer>
  );
}

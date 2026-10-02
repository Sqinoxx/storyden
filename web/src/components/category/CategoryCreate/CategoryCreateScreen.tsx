import { AssetUploadEditor } from "@/components/asset/AssetUploadEditor/AssetUploadEditor";
import { CategoryTreeSelect } from "@/components/category/CategoryTreeSelect/CategoryTreeSelect";
import { ColourPickerField } from "@/components/ui/ColourPickerField";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormFeedback } from "@/components/ui/form/FormFeedback";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Input, InputPrefix } from "@/components/ui/input";
import { WEB_ADDRESS } from "@/config";
import {
  CATEGORY_COVER_HEIGHT,
  CATEGORY_COVER_WIDTH,
} from "@/lib/category/cover";
import { useTranslation } from "@/lib/i18n";
import { HStack, VStack, WStack, styled } from "@/styled-system/jsx";

import { CategoryCreateProps, useCategoryCreate } from "./useCategoryCreate";

export type { CategoryCreateProps };

export function CategoryCreateScreen(props: CategoryCreateProps) {
  const { register, onSubmit, control, formState, handleImageUpload } =
    useCategoryCreate(props);
  const t = useTranslation();

  const hostname = new URL(WEB_ADDRESS).host;

  return (
    <VStack alignItems="start" gap="4">
      <styled.p>{t.category.createIntro}</styled.p>
      <styled.form
        display="flex"
        flexDir="column"
        gap="4"
        w="full"
        onSubmit={onSubmit}
      >
        <FormControl>
          <FormLabel>{t.category.coverImage}</FormLabel>
          <AssetUploadEditor
            width={CATEGORY_COVER_WIDTH}
            height={CATEGORY_COVER_HEIGHT}
            onUpload={handleImageUpload}
          />
          <FormHelperText>{t.category.coverImageHelper}</FormHelperText>
        </FormControl>

        <FormControl>
          <FormLabel>{t.category.name}</FormLabel>
          <Input {...register("name")} type="text" />
          <FormHelperText>{t.category.nameHelper}</FormHelperText>
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
              {...register("slug")}
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
          <FormFeedback error={formState.errors["slug"]?.message}>
            {t.category.slugHelper}
          </FormFeedback>
        </FormControl>

        <FormControl>
          <FormLabel>{t.category.description}</FormLabel>

          {/* TODO: Make a larger textarea component for this. */}
          <Input {...register("description")} type="text" />
          <FormHelperText>{t.category.descriptionHelper}</FormHelperText>
        </FormControl>

        <FormControl>
          <FormLabel>{t.category.parentCategory}</FormLabel>
          <CategoryTreeSelect
            name="parent"
            control={control}
            selectable="all"
            emptyOption={{ label: t.category.noParentCategory, value: "" }}
          />
          <FormHelperText>{t.category.parentHelper}</FormHelperText>
        </FormControl>

        <FormControl>
          <FormLabel>{t.category.colour}</FormLabel>
          <ColourPickerField control={control} name="colour" />
          <FormHelperText>{t.category.colourHelper}</FormHelperText>
        </FormControl>

        <WStack>
          <Button flexGrow="1" type="button" onClick={props.onClose}>
            {t.common.cancel}
          </Button>
          <Button flexGrow="1" type="submit">
            {t.actions.create}
          </Button>
        </WStack>
      </styled.form>
    </VStack>
  );
}

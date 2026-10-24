import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useMemo } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import {
  categoryCreate,
  useCategoryList,
} from "@/api/openapi-client/categories";
import { Asset } from "@/api/openapi-schema";
import { UseDisclosureProps } from "@/utils/useDisclosure";

import { handle } from "@/api/client";
import {
  categoryDescriptionSchema,
  categoryNameSchema,
  categorySlugSchema,
  findSiblingWithName,
  normaliseCategoryFields,
} from "@/lib/category/form";
import { useCategoryMutations } from "@/lib/category/mutation";
import { useTranslation } from "@/lib/i18n";
import { slugify } from "@/utils/slugify";

function makeFormSchema(nameRequired: string) {
  return z.object({
    name: categoryNameSchema(nameRequired),
    slug: categorySlugSchema,
    description: categoryDescriptionSchema,
    colour: z.string().default("#8577ce"),
    parent: z.string().optional(),
    cover_image_asset_id: z.string().optional(),
  });
}
type Form = z.infer<ReturnType<typeof makeFormSchema>>;

export interface CategoryCreateProps extends UseDisclosureProps {
  defaultParent?: string;
}

export function useCategoryCreate(props: CategoryCreateProps) {
  const t = useTranslation();
  const { revalidateList } = useCategoryMutations();
  const { data: categoryList } = useCategoryList();

  const schema = useMemo(
    () => makeFormSchema(t.category.nameRequired),
    [t.category.nameRequired],
  );

  const {
    register,
    handleSubmit,
    control,
    setValue,
    setError,
    watch,
    formState,
  } = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: {
      colour: "#8577ce",
      parent: props.defaultParent,
    },
  });

  const name = watch("name");
  const slugEdited = formState.dirtyFields.slug;

  useEffect(() => {
    if (slugEdited) return;
    setValue("slug", slugify(name ?? ""));
  }, [name, slugEdited, setValue]);

  const onSubmit = handleSubmit(async (raw) => {
    const data = normaliseCategoryFields(raw);
    const sibling = findSiblingWithName(
      categoryList?.categories,
      data.name,
      data.parent,
    );
    if (sibling) {
      setError("name", { message: t.category.nameTakenInParent });
      return;
    }

    await handle(async () => {
      await categoryCreate({
        ...data,
        slug: data.slug || undefined,
        parent: data.parent || undefined,
      });
      props.onClose?.();
      await revalidateList();
    });
  });

  function handleImageUpload(asset: Asset) {
    setValue("cover_image_asset_id", asset.id);
  }

  return {
    onSubmit,
    register,
    control,
    formState,
    handleImageUpload,
  };
}

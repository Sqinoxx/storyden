import { zodResolver } from "@hookform/resolvers/zod";
import { usePathname, useRouter } from "next/navigation";
import { useMemo } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { useCategoryList } from "@/api/openapi-client/categories";
import { Asset, Category } from "@/api/openapi-schema";
import {
  categoryDescriptionSchema,
  categoryNameSchema,
  categorySlugSchema,
  findSiblingWithName,
  normaliseCategoryFields,
} from "@/lib/category/form";
import { useCategoryMutations } from "@/lib/category/mutation";
import { useTranslation } from "@/lib/i18n";
import { UseDisclosureProps } from "@/utils/useDisclosure";

export type Props = {
  category: Category;
} & UseDisclosureProps;

function makeFormSchema(nameRequired: string) {
  return z.object({
    name: categoryNameSchema(nameRequired),
    slug: categorySlugSchema,
    description: categoryDescriptionSchema,
    colour: z.string().default("#fff"),
    cover_image: z.custom<Asset>().nullable().optional(),
    related: z.array(z.string()).default([]),
  });
}
export type Form = z.infer<ReturnType<typeof makeFormSchema>>;

export function useCategoryEdit(props: Props) {
  const t = useTranslation();
  const { data: categoryList } = useCategoryList();

  const schema = useMemo(
    () => makeFormSchema(t.category.nameRequired),
    [t.category.nameRequired],
  );

  const form = useForm<Form>({
    resolver: zodResolver(schema),
    defaultValues: {
      name: props.category.name,
      slug: props.category.slug,
      description: props.category.description,
      colour: props.category.colour,
      cover_image: props.category.cover_image || null,
      related: props.category.related?.map((c) => c.id) ?? [],
    },
  });
  const pathname = usePathname();
  const router = useRouter();

  const { revalidateList, updateCategory } = useCategoryMutations();

  const handleSubmit = form.handleSubmit(async (raw) => {
    const data = normaliseCategoryFields(raw);
    const originalSlug = props.category.slug;

    const sibling = findSiblingWithName(
      categoryList?.categories,
      data.name,
      props.category.parent,
      props.category.id,
    );
    if (sibling) {
      form.setError("name", { message: t.category.nameTakenInParent });
      return;
    }

    await handle(
      async () => {
        const { cover_image, slug, ...rest } = data;
        const updated = await updateCategory(originalSlug, {
          ...rest,
          slug: slug || originalSlug,
          cover_image_asset_id: cover_image?.id || null,
        });

        props.onClose?.();

        if (updated.slug !== originalSlug) {
          if (pathname.includes(`/d/${originalSlug}`)) {
            router.replace(`/d/${updated.slug}`);
          }
        }
      },
      {
        promiseToast: {
          loading: t.category.updating,
          success: t.category.updated,
        },
        cleanup: () => revalidateList(),
      },
    );
  });

  function handleCancel() {
    form.reset();
    props.onClose?.();
  }

  function handleImageUpload(asset: Asset) {
    form.setValue("cover_image", asset);
  }

  function handleImageRemove() {
    form.setValue("cover_image", null);
  }

  return {
    form,
    handlers: {
      handleSubmit,
      handleCancel,
      handleImageUpload,
      handleImageRemove,
    },
  };
}

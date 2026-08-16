import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { mutate } from "swr";
import { z } from "zod";

import {
  categoryCreate,
  getCategoryListKey,
} from "@/api/openapi-client/categories";
import { Asset } from "@/api/openapi-schema";
import { UseDisclosureProps } from "@/utils/useDisclosure";

import { handle } from "@/api/client";
import { isSlug } from "@/utils/slugify";

export const FormSchema = z.object({
  name: z.string().min(1, "Please enter a name for the category."),
  slug: z
    .string()
    .min(1, "Please enter a URL slug for the category.")
    .refine(isSlug, "The slug must only contain letters, numbers, hyphens and underscores."),
  description: z.string().min(1, "Please enter a short description."),
  colour: z.string().default("#8577ce"),
  parent: z.string().optional(),
  cover_image_asset_id: z.string().optional(),
});
export type Form = z.infer<typeof FormSchema>;

export interface CategoryCreateProps extends UseDisclosureProps {
  defaultParent?: string;
}

export function useCategoryCreate(props: CategoryCreateProps) {
  const { register, handleSubmit, control, setValue, formState } = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      colour: "#8577ce",
      parent: props.defaultParent,
    },
  });

  const onSubmit = handleSubmit(async (data) => {
    await handle(async () => {
      await categoryCreate(data);
      props.onClose?.();
      mutate(getCategoryListKey());
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

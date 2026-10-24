import { z } from "zod";

import { Category } from "@/api/openapi-schema";
import { slugify } from "@/utils/slugify";

export function categoryNameSchema(requiredMessage: string) {
  return z.string().trim().min(1, requiredMessage);
}

export const categorySlugSchema = z.string().optional();

export const categoryDescriptionSchema = z.string().optional();

export function normaliseCategoryFields<
  T extends { name: string; slug?: string; description?: string },
>(data: T) {
  return {
    ...data,
    name: data.name.trim(),
    slug: slugify(data.slug ?? ""),
    description: data.description?.trim() ?? "",
  };
}

function normaliseName(name: string) {
  return name.trim().toLocaleLowerCase();
}

export function findSiblingWithName(
  categories: Category[] | undefined,
  name: string,
  parent: string | undefined,
  exclude?: string,
) {
  const target = normaliseName(name);

  return categories?.find(
    (c) =>
      c.id !== exclude &&
      (c.parent || undefined) === (parent || undefined) &&
      normaliseName(c.name) === target,
  );
}

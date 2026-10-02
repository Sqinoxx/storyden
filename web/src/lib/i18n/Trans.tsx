"use client";

import { useTranslation } from "./LanguageContext";
import { Translations } from "./translations/en";

type Leaves<T, P extends string = ""> = {
  [K in keyof T & string]: T[K] extends string
    ? `${P}${K}`
    : Leaves<T[K], `${P}${K}.`>;
}[keyof T & string];

export type TranslationPath = Leaves<Translations>;

type Props = {
  path: TranslationPath;
  vars?: Record<string, string | number>;
};

export function Trans({ path, vars }: Props) {
  const t = useTranslation();

  const value = path
    .split(".")
    .reduce<unknown>(
      (node, key) => (node as Record<string, unknown>)[key],
      t,
    ) as string;

  if (!vars) return <>{value}</>;

  return (
    <>
      {Object.entries(vars).reduce(
        (text, [key, v]) => text.replaceAll(`{${key}}`, String(v)),
        value,
      )}
    </>
  );
}

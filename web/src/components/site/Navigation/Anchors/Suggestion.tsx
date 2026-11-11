"use client";

import { LightbulbIcon } from "@/components/ui/icons/Lightbulb";
import { Item } from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";

export const SuggestionID = "suggestion";

export function SuggestionMenuItem() {
  const t = useTranslation();
  return (
    <Item value={SuggestionID}>
      <LightbulbIcon />
      &nbsp;<span>{t.nav.suggestImprovement}</span>
    </Item>
  );
}

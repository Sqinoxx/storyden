"use client";

import { useLanguage } from "@/lib/i18n";
import { Item } from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";

export const LanguageID = "language";

export function LanguageMenuItem() {
  const { language, setLanguage } = useLanguage();
  const t = useTranslation();

  function handleClick() {
    setLanguage(language === "en" ? "de" : "en");
  }

  const flagEmoji = language === "de" ? "🇬🇧" : "🇩🇪";
  const nextLangLabel = language === "de" ? "English" : "Deutsch";

  return (
    <Item value={LanguageID} asChild>
      <button
        type="button"
        onClick={handleClick}
        title={t.nav.switchLanguage}
        style={{ cursor: "pointer", width: "100%" }}
      >
        <span style={{ fontSize: "1rem", lineHeight: 1 }}>{flagEmoji}</span>
        &nbsp;<span>{nextLangLabel}</span>
      </button>
    </Item>
  );
}

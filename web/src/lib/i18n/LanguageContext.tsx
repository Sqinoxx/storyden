"use client";

import {
  ReactNode,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";

import { de } from "./translations/de";
import { Translations, en } from "./translations/en";

export type Language = "en" | "de";

const STORAGE_KEY = "storyden-language";

// Translation map per supported language
const translations: Record<Language, Translations> = { en, de };

type LanguageContextValue = {
  language: Language;
  t: Translations;
  setLanguage: (lang: Language) => void;
};

const LanguageContext = createContext<LanguageContextValue>({
  language: "de",
  t: de,
  setLanguage: () => {},
});

function getInitialLanguage(): Language {
  if (typeof window === "undefined") return "de";
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "en" || stored === "de") return stored;
  // Use browser language as fallback
  const browserLang = navigator.language.slice(0, 2).toLowerCase();
  return browserLang === "de" ? "de" : "en";
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>("de");

  useEffect(() => {
    setLanguageState(getInitialLanguage());
  }, []);

  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  const setLanguage = useCallback((lang: Language) => {
    setLanguageState(lang);
    localStorage.setItem(STORAGE_KEY, lang);
  }, []);

  return (
    <LanguageContext.Provider
      value={{ language, t: translations[language], setLanguage }}
    >
      {children}
    </LanguageContext.Provider>
  );
}

export function useLanguage() {
  return useContext(LanguageContext);
}

/** Shortcut: only returns the translation object */
export function useTranslation() {
  return useContext(LanguageContext).t;
}

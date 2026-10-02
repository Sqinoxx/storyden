import { useCallback } from "react";

import { useTranslation } from "./LanguageContext";
import { en } from "./translations/en";

type ValidationKey = keyof typeof en.validation;

const keyByMessage = new Map(
  Object.entries(en.validation).map(([key, message]) => [
    message,
    key as ValidationKey,
  ]),
);

export function useTranslateMessage() {
  const t = useTranslation();

  return useCallback(
    (message: string) => {
      const key = keyByMessage.get(message);
      return key ? t.validation[key] : message;
    },
    [t],
  );
}

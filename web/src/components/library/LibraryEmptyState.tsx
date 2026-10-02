import { useTranslation } from "@/lib/i18n";

import { EmptyState } from "../site/EmptyState";

export function LibraryEmptyState() {
  const t = useTranslation();
  return (
    <EmptyState w="full">
      <p>{t.library.empty}</p>
    </EmptyState>
  );
}

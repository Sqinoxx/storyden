"use client";

import { LinkButton } from "@/components/ui/link-button";
import { JsxStyleProps } from "@/styled-system/types";
import { useTranslation } from "@/lib/i18n";

export function LoginAnchor(props: JsxStyleProps) {
  const t = useTranslation();
  return (
    <LinkButton href="/login" variant="ghost" size="sm" {...props}>
      {t.nav.login}
    </LinkButton>
  );
}

export function RegisterAnchor(props: JsxStyleProps) {
  return (
    <LinkButton href="/register" size="sm" {...props}>
      Register
    </LinkButton>
  );
}

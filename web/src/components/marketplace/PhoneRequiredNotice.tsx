import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { CardBox, LStack, styled } from "@/styled-system/jsx";

export const PHONE_SETTINGS_URL = "/settings?tab=account#phone";

export function PhoneRequiredNotice() {
  const t = useTranslation();

  return (
    <CardBox w="full" data-testid="phone-required">
      <LStack gap="2">
        <styled.h2 fontWeight="semibold">
          {t.marketplace.phoneRequired.title}
        </styled.h2>
        <styled.p color="fg.muted" fontSize="sm">
          {t.marketplace.phoneRequired.description}
        </styled.p>
        <LinkButton href={PHONE_SETTINGS_URL} size="sm">
          {t.marketplace.phoneRequired.action}
        </LinkButton>
      </LStack>
    </CardBox>
  );
}

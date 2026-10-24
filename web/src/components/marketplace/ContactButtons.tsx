import * as Clipboard from "@/components/ui/clipboard";
import { IconButton } from "@/components/ui/icon-button";
import { CheckIcon } from "@/components/ui/icons/Check";
import { CopyIcon } from "@/components/ui/icons/Copy";
import { WhatsAppIcon } from "@/components/ui/icons/WhatsApp";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { telURL, whatsappURL } from "@/lib/marketplace/marketplace";
import { HStack, VStack, styled } from "@/styled-system/jsx";

type Props = {
  phone: string;
  message: string;
};

export function ContactButtons({ phone, message }: Props) {
  const t = useTranslation();

  return (
    <VStack gap="2" alignItems="start" w="full">
      <LinkButton
        href={whatsappURL(phone, message)}
        size="sm"
        colorPalette="green"
      >
        <WhatsAppIcon />
        {t.marketplace.contact.whatsapp}
      </LinkButton>
      <HStack gap="1">
        <styled.a
          href={telURL(phone)}
          fontSize="sm"
          fontFamily="mono"
          color="fg.default"
          textDecoration="underline"
          data-testid="contact-phone"
        >
          {phone}
        </styled.a>
        <Clipboard.Root value={phone}>
          <Clipboard.Trigger asChild>
            <IconButton
              size="xs"
              variant="ghost"
              aria-label={t.marketplace.contact.copyPhone}
              title={t.marketplace.contact.copyPhone}
            >
              <Clipboard.Indicator copied={<CheckIcon />}>
                <CopyIcon />
              </Clipboard.Indicator>
            </IconButton>
          </Clipboard.Trigger>
        </Clipboard.Root>
      </HStack>
    </VStack>
  );
}

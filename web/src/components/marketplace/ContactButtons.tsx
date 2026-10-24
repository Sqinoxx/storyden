import { WhatsAppIcon } from "@/components/ui/icons/WhatsApp";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { telURL, whatsappURL } from "@/lib/marketplace/marketplace";
import { HStack, styled } from "@/styled-system/jsx";

type Props = {
  phone: string;
  message: string;
};

export function ContactButtons({ phone, message }: Props) {
  const t = useTranslation();

  return (
    <HStack gap="2" flexWrap="wrap" w="full">
      <LinkButton
        href={whatsappURL(phone, message)}
        size="sm"
        colorPalette="green"
      >
        <WhatsAppIcon />
        {t.marketplace.contact.whatsapp}
      </LinkButton>
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
    </HStack>
  );
}

import { formatDistanceToNow } from "date-fns";

import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { useLanguage, useTranslation } from "@/lib/i18n";
import { HStack, VStack, styled } from "@/styled-system/jsx";
import { dateFnsLocale } from "@/utils/date";

import { DeleteDeviceTrigger } from "./DeleteDevice/DeleteDeviceTrigger";
import { Props, useDevices } from "./useDevices";

export function Devices(props: Props) {
  const t = useTranslation();
  const { handleDeviceRegister } = useDevices();
  const { language } = useLanguage();

  return (
    <VStack w="full" alignItems="start">
      <Heading size="sm">{t.settings.devices.title}</Heading>

      <p>{t.settings.devices.description}</p>

      <styled.ul w="full" display="flex" flexDir="column" gap="2">
        {props.active.map((v) => (
          <styled.li
            key={v.id}
            display="flex"
            flexDir="column"
            borderColor="border.muted"
            borderWidth="thin"
            borderRadius="md"
            p="2"
            gap="2"
            minW="0"
          >
            <HStack justify="space-between">
              <Heading size="xs">{v.name}</Heading>
            </HStack>

            <styled.p
              minW="0"
              className="typography"
              whiteSpace="nowrap"
              textOverflow="ellipsis"
              overflow="hidden"
            >
              {t.settings.devices.deviceId}{" "}
              <styled.code title={v.identifier}> {v.identifier}</styled.code>
            </styled.p>

            <HStack justify="space-between">
              <styled.p>
                {t.settings.devices.created}{" "}
                <time>
                  {formatDistanceToNow(new Date(v.created_at), {
                    addSuffix: true,
                    locale: dateFnsLocale(language),
                  })}
                </time>
              </styled.p>

              <DeleteDeviceTrigger id={v.id} />
            </HStack>
          </styled.li>
        ))}
      </styled.ul>

      <Button variant="subtle" onClick={handleDeviceRegister}>
        {t.settings.devices.register}
      </Button>
    </VStack>
  );
}

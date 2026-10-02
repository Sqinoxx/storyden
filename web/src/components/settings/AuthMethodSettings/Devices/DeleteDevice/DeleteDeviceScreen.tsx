import { Button } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { HStack, VStack } from "@/styled-system/jsx";

import {
  Props,
  WithDisclosure,
  useDeleteDeviceScreen,
} from "./useDeleteDeviceScreen";

export function DeleteDeviceScreen(props: WithDisclosure<Props>) {
  const t = useTranslation();
  const { handleConfirm } = useDeleteDeviceScreen(props);

  return (
    <VStack maxW="prose">
      <p>{t.settings.devices.deleteWarning}</p>
      <HStack
        w="full"
        justifyContent="space-between"
        alignItems="center"
        justify="end"
        pb="3"
        gap="4"
      >
        <Button flexGrow="1" size="sm" variant="ghost" onClick={props.onClose}>
          {t.common.cancel}
        </Button>
        <Button
          flexGrow="1"
          size="sm"
          colorPalette="red"
          onClick={handleConfirm}
        >
          {t.common.delete}
        </Button>
      </HStack>
    </VStack>
  );
}

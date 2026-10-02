import { PropsWithChildren } from "react";

import { useDisclosure } from "@/utils/useDisclosure";

import { Button } from "@/components/ui/button";

import { DeleteDeviceModal } from "./DeleteDeviceModal";
import { Props } from "./useDeleteDeviceScreen";
import { useTranslation } from "@/lib/i18n";

export function DeleteDeviceTrigger(props: PropsWithChildren<Props>) {
  const t = useTranslation();
  const { onOpen, isOpen, onClose } = useDisclosure();
  return (
    <>
      <Button size="xs" colorPalette="red" onClick={onOpen}>
        {props.children ?? t.common.delete}
      </Button>
      <DeleteDeviceModal isOpen={isOpen} onClose={onClose} {...props} />
    </>
  );
}

import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { useTranslation } from "@/lib/i18n";

import { DeleteDeviceScreen } from "./DeleteDeviceScreen";
import { Props, WithDisclosure } from "./useDeleteDeviceScreen";

export function DeleteDeviceModal(props: WithDisclosure<Props>) {
  const t = useTranslation();
  return (
    <>
      <ModalDrawer
        isOpen={props.isOpen}
        onClose={props.onClose}
        title={t.settings.devices.deleteTitle}
      >
        <DeleteDeviceScreen onClose={props.onClose} id={props.id} />
      </ModalDrawer>
    </>
  );
}

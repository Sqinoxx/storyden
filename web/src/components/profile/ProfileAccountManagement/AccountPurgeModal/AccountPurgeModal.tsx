import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { useTranslation } from "@/lib/i18n";
import { UseDisclosureProps } from "@/utils/useDisclosure";

import { AccountPurgeScreen } from "./AccountPurgeScreen";
import { Props } from "./useAccountPurge";

export function AccountPurgeModal({
  accountId,
  handle,
  onClose,
  onOpen,
  isOpen,
}: UseDisclosureProps & Props) {
  const t = useTranslation();
  return (
    <ModalDrawer
      onOpen={onOpen}
      isOpen={isOpen}
      onClose={onClose}
      title={t.profile.purgeTitle.replace("{handle}", handle)}
    >
      <AccountPurgeScreen
        accountId={accountId}
        handle={handle}
        onSave={onClose}
      />
    </ModalDrawer>
  );
}

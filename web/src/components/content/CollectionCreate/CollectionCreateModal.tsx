import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { useTranslation } from "@/lib/i18n";

import { CollectionCreateScreen } from "./CollectionCreateScreen";
import { Props } from "./useCollectionCreate";

export function CollectionCreateModal({ session, ...props }: Props) {
  const t = useTranslation();
  return (
    <>
      <ModalDrawer
        isOpen={props.isOpen}
        onClose={props.onClose}
        title={t.collections.createTitle}
      >
        <CollectionCreateScreen
          id={props.id}
          session={session}
          onClose={props.onClose}
        />
      </ModalDrawer>
    </>
  );
}

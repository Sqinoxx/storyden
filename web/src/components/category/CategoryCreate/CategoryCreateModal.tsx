import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { useTranslation } from "@/lib/i18n";

import {
  CategoryCreateProps,
  CategoryCreateScreen,
} from "./CategoryCreateScreen";

export function CategoryCreateModal(props: CategoryCreateProps) {
  const t = useTranslation();
  return (
    <>
      <ModalDrawer
        isOpen={props.isOpen}
        onClose={props.onClose}
        onOpenChange={props.onOpenChange}
        title={t.category.createTitle}
      >
        <CategoryCreateScreen {...props} />
      </ModalDrawer>
    </>
  );
}

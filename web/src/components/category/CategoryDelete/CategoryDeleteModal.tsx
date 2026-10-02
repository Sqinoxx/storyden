import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";

import { CategoryDeleteProps, CategoryDeleteScreen } from "./CategoryDeleteScreen";
import { useTranslation } from "@/lib/i18n";

export function CategoryDeleteModal(props: CategoryDeleteProps) {
  const t = useTranslation();
  return (
    <ModalDrawer
      isOpen={props.isOpen}
      onClose={props.onClose}
      title={t.category.deleteTitle}
    >
      <CategoryDeleteScreen {...props} />
    </ModalDrawer>
  );
}
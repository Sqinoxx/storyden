import { ButtonProps } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { MoreIcon } from "@/components/ui/icons/More";
import { useTranslation } from "@/lib/i18n";

export function MoreAction(props: ButtonProps) {
  const t = useTranslation();
  return (
    <IconButton variant="ghost" aria-label={t.common.moreOptions} {...props}>
      <MoreIcon />
    </IconButton>
  );
}

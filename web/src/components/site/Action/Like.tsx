import { ButtonProps } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { LikeIcon, LikeSavedIcon } from "@/components/ui/icons/Like";
import { useTranslation } from "@/lib/i18n";

type Props = ButtonProps & { liked: boolean };

export function LikeAction(props: Props) {
  const { liked, ...rest } = props;
  const t = useTranslation();
  return (
    <IconButton
      variant="subtle"
      size="xs"
      aria-pressed={liked}
      aria-label={liked ? t.thread.unlikeAction : t.thread.likeAction}
      title={liked ? t.thread.unlikeAction : t.thread.likeAction}
      {...rest}
    >
      {liked ? <LikeSavedIcon /> : <LikeIcon />}
    </IconButton>
  );
}

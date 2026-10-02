import { LikeAction } from "@/components/site/Action/Like";
import { Button } from "@/components/ui/button";
import { LikeIcon, LikeSavedIcon } from "@/components/ui/icons/Like";
import { useTranslation } from "@/lib/i18n";
import { styled } from "@/styled-system/jsx";

import { Props, useLikeButton } from "./useLikeButton";

type LikeButtonProps = Props & {
  showCount?: boolean;
};

export function LikeButton({ showCount = false, ...props }: LikeButtonProps) {
  const t = useTranslation();
  const { enabled, handleClick } = useLikeButton({ thread: props.thread });
  const likeCount = props.thread.likes.likes;

  if (showCount) {
    return (
      <Button
        type="button"
        variant="subtle"
        display="flex"
        size="xs"
        gap="1"
        color="fg.muted"
        aria-label={
          props.thread.likes.liked ? t.thread.unlikeAction : t.thread.likeAction
        }
        title={
          props.thread.likes.liked ? t.thread.unlikeAction : t.thread.likeAction
        }
        onClick={handleClick}
        disabled={!enabled}
      >
        <span>
          {props.thread.likes.liked ? (
            <LikeSavedIcon width="4" />
          ) : (
            <LikeIcon width="4" />
          )}
        </span>
        <styled.span
          fontSize="sm"
          fontWeight="medium"
          fontVariantNumeric="tabular-nums"
          fontVariant="tabular-nums"
        >
          {likeCount}
        </styled.span>
      </Button>
    );
  }

  return (
    <LikeAction
      variant="subtle"
      size="xs"
      liked={props.thread.likes.liked}
      onClick={handleClick}
      disabled={!enabled}
    />
  );
}

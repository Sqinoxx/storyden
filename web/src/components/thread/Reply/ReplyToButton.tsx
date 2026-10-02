import { Reply, Thread } from "@/api/openapi-schema";
import { IconButton } from "@/components/ui/icon-button";
import { ReplyIcon } from "@/components/ui/icons/Reply";
import { useTranslation } from "@/lib/i18n";

import { useReplyContext } from "../ReplyContext";

type Props = {
  thread: Thread;
  reply: Reply;
};

export function ReplyToButton(props: Props) {
  const { setReplyTo } = useReplyContext();
  const t = useTranslation();

  function handleClick() {
    setReplyTo(props.thread, props.reply);
  }

  return (
    <IconButton
      type="button"
      size="xs"
      variant="ghost"
      aria-label={t.thread.replyToThis}
      onClick={handleClick}
    >
      <ReplyIcon />
    </IconButton>
  );
}

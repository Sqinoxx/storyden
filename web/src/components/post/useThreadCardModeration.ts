import { useRouter } from "next/navigation";

import { handle } from "@/api/client";
import { ThreadReference } from "@/api/openapi-schema";
import { useConfirmation } from "@/components/site/useConfirmation";
import { useFeedMutations } from "@/lib/feed/mutation";
import { useTranslation } from "@/lib/i18n";
import { withUndo } from "@/lib/thread/undo";

export function useThreadCardModeration(thread: ThreadReference) {
  const t = useTranslation();
  const router = useRouter();
  const { updateThread, deleteThread, revalidate } = useFeedMutations();

  const {
    isConfirming: isConfirmingDelete,
    handleConfirmAction: handleConfirmDelete,
    handleCancelAction: handleCancelDelete,
  } = useConfirmation(handleDelete);

  async function handleAcceptThread() {
    await handle(
      async () => {
        await updateThread(thread.id, { visibility: "published" });
      },
      {
        promiseToast: {
          loading: t.toasts.accepting,
          success: t.toasts.threadAccepted,
        },
        cleanup: async () => {
          await revalidate();
        },
      },
    );
  }

  function handleEditAndAccept() {
    router.push(`/t/${thread.slug}?edit=true`);
  }

  async function handleDelete() {
    await handle(
      async () => {
        await withUndo({
          message: t.toasts.threadDeleted,
          duration: 5000,
          toastId: `thread-${thread.id}`,
          action: async () => {
            await deleteThread(thread.id);
          },
          onUndo: () => {},
        });
      },
      {
        cleanup: async () => {
          await revalidate();
        },
      },
    );
  }

  return {
    isConfirmingDelete,
    handlers: {
      handleAcceptThread,
      handleEditAndAccept,
      handleConfirmDelete,
      handleCancelDelete,
    },
  };
}

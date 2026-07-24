"use client";

import { handle } from "@/api/client";
import { ButtonProps } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { CreateIcon } from "@/components/ui/icons/Create";
import { Item } from "@/components/ui/menu";
import { useLibraryMutation } from "@/lib/library/library";
import { useTranslation } from "@/lib/i18n";

type Props = ButtonProps & {
  parentSlug?: string;
  hideLabel?: boolean;
  disableRedirect?: boolean;
  onComplete?: () => void;
};

export const CreatePageID = "create-page";
export const CreatePageLabel = "Create";
export const CreatePageIcon = <CreateIcon />;

export function CreatePageAction({
  parentSlug,
  hideLabel,
  disableRedirect,
  onComplete,
  ...props
}: Props) {
  const { createNode, revalidate } = useLibraryMutation();
  const t = useTranslation();

  async function handleCreate() {
    await handle(
      async () => {
        await createNode({ parentSlug, disableRedirect });
      },
      {
        promiseToast: {
          loading: t.library.creatingPage,
          success: t.library.pageCreated,
        },
        cleanup: async () => {
          await revalidate();
          onComplete?.();
        },
      },
    );
  }

  return (
    <IconButton
      type="button"
      size="xs"
      variant="subtle"
      px={hideLabel ? "0" : "1"}
      onClick={handleCreate}
      {...props}
    >
      {CreatePageIcon}
      {!hideLabel && (
        <>
          <span>{t.actions.create}</span>
        </>
      )}
    </IconButton>
  );
}

export function CreatePageMenuItem({ hideLabel }: Props) {
  const t = useTranslation();
  return (
    <Item value={CreatePageID}>
      {CreatePageIcon}
      {!hideLabel && (
        <>
          &nbsp;<span>{t.actions.create}</span>
        </>
      )}
    </Item>
  );
}

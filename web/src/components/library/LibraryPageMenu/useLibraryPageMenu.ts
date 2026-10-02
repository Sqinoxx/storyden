import { MenuSelectionDetails } from "@ark-ui/react";
import { match } from "ts-pattern";

import {
  Account,
  Identifier,
  Node,
  Permission,
  Visibility,
} from "@/api/openapi-schema";
import { useSession } from "@/auth";

import { handle } from "@/api/client";
import { useConfirmation } from "@/components/site/useConfirmation";
import { useLibraryMutation } from "@/lib/library/library";
import { hasPermission } from "@/utils/permissions";
import { useTranslation } from "@/lib/i18n";

export type Props = {
  node: Node;
  parentID?: Identifier;
  open?: boolean;
  onClose?: () => void;
};

export function useLibraryPageMenu(props: Props) {
  const t = useTranslation();
  const account = useSession();
  const {
    deleteNode,
    updateNodeVisibility,
    updateNodeChildVisibility,
    revalidate,
  } = useLibraryMutation(props.node);

  const {
    isConfirming: isConfirmingDelete,
    handleConfirmAction: handleConfirmDelete,
    handleCancelAction: handleCancelDelete,
  } = useConfirmation(handleDelete);

  const isManager = hasPermission(account, Permission.MANAGE_LIBRARY);
  const isOwner = account?.id === props.node.owner.id;

  const availableOperations = visibilityStateChanges[
    props.node.visibility
  ].filter((c) => account && c.condition(account, props.node));

  // Managers can delete any page, owners can only delete non-published pages.
  const deleteEnabled =
    isManager || (isOwner && props.node.visibility !== Visibility.published);

  const isChildrenHidden = props.node.hide_child_tree;

  async function handleToggleChildrenVisibility() {
    await handle(
      async () => {
        await updateNodeChildVisibility(props.node.slug, !isChildrenHidden);
      },
      {
        promiseToast: {
          loading: t.toasts.saving,
          success: match(!isChildrenHidden)
            .with(true, () => t.library.childrenHidden)
            .with(false, () => t.library.childrenVisible)
            .exhaustive(),
        },
      },
    );
  }

  async function handleDelete() {
    return handle(
      async () => {
        await deleteNode(props.node.slug, props.parentID);
      },
      {
        cleanup: async () => await revalidate(),
      },
    );
  }

  async function handleVisibilityChange(visibility: Visibility) {
    await handle(
      async () => {
        await updateNodeVisibility(props.node.slug, visibility, props.parentID);
      },
      {
        promiseToast: {
          loading: t.toasts.saving,
          success: match(visibility)
            .with(Visibility.published, () => t.library.toastPublished)
            .with(Visibility.draft, () => t.library.toastDraft)
            .with(Visibility.review, () => t.library.toastReview)
            .with(Visibility.unlisted, () => t.library.toastUnlisted)
            .exhaustive(),
        },
        cleanup: () => revalidate(),
      },
    );
  }

  function handleSelect({ value }: MenuSelectionDetails) {
    if (value === "report-node") {
      return;
    }

    switch (value as Visibility | "toggle-hide-in-tree" | "delete") {
      case "toggle-hide-in-tree":
        return handleToggleChildrenVisibility();

      case "delete":
        return handleConfirmDelete();

      case Visibility.draft:
        return handleVisibilityChange(Visibility.draft);

      case Visibility.unlisted:
        return handleVisibilityChange(Visibility.unlisted);

      case Visibility.review:
        return handleVisibilityChange(Visibility.review);

      case Visibility.published:
        return handleVisibilityChange(Visibility.published);
    }
  }

  return {
    availableOperations,
    deleteEnabled,
    isConfirmingDelete,
    isChildrenHidden,
    isManager,
    handlers: {
      handleCancelDelete,
      handleSelect,
    },
  };
}

type VisibilityStateChangeMenuItem = {
  label: string;
  targetVisibility: Visibility;
  condition: (account: Account, node: Node) => boolean;
};

const visibilityStateChanges: Record<
  Visibility,
  VisibilityStateChangeMenuItem[]
> = {
  [Visibility.draft]: [
    {
      label: "Publish to library",
      targetVisibility: Visibility.published,
      condition: (account) => hasPermission(account, Permission.MANAGE_LIBRARY),
    },
    {
      label: "Submit for review",
      targetVisibility: Visibility.review,
      condition: (account, node) => account.id === node.owner.id,
    },
    {
      label: "Publish to profile",
      targetVisibility: Visibility.unlisted,
      condition: (account, node) => account.id === node.owner.id,
    },
  ],
  [Visibility.unlisted]: [
    {
      label: "Revert to draft",
      targetVisibility: Visibility.draft,
      condition: (account, node) => account.id === node.owner.id,
    },
    {
      label: "Submit for review",
      targetVisibility: Visibility.review,
      condition: (account, node) => account.id === node.owner.id,
    },
  ],
  [Visibility.review]: [
    {
      label: "Publish to library",
      targetVisibility: Visibility.published,
      condition: (account) => hasPermission(account, Permission.MANAGE_LIBRARY),
    },
    {
      label: "Reject",
      targetVisibility: Visibility.draft,
      condition: (account, node) =>
        account.id !== node.owner.id &&
        hasPermission(account, Permission.MANAGE_LIBRARY),
    },
    {
      label: "Revert to draft",
      targetVisibility: Visibility.draft,
      condition: (account, node) => account.id === node.owner.id,
    },
  ],
  [Visibility.published]: [
    {
      label: "Unpublish",
      targetVisibility: Visibility.draft,
      condition: (account) => hasPermission(account, Permission.MANAGE_LIBRARY),
    },
  ],
};

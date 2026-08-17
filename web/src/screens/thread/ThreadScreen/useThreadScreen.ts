import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { parseAsBoolean, parseAsString, useQueryState } from "nuqs";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { threadUpdate, useThreadGet } from "@/api/openapi-client/threads";
import {
  Account,
  Asset,
  DatagraphItemKind,
  Permission,
  ThreadGetResponse,
} from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { useConfirmation } from "@/components/site/useConfirmation";
import { extractDocumentAssetsFromThread } from "@/components/post/ThreadCard";
import { useBeacon } from "@/lib/beacon/useBeacon";
import { useReportContext } from "@/lib/report/useReportContext";
import type { Settings, SignatureConfig } from "@/lib/settings/settings";
import { useThreadMutations } from "@/lib/thread/mutation";
import { withUndo } from "@/lib/thread/undo";
import {
  getCleanFilename,
  normalizeAssetPath,
  normalizeFilename,
} from "@/utils/asset";
import { hasPermission } from "@/utils/permissions";

export type Props = {
  initialSession?: Account;
  initialPage?: number;
  slug: string;
  thread: ThreadGetResponse;
  initialSettings?: Settings;
  initialSignatureConfig?: SignatureConfig;
};

export const FormSchema = z.object({
  title: z.string().min(1, "Please enter a title."),
  body: z.string().min(1),
  tags: z.array(z.string()).optional(),
});
export type Form = z.infer<typeof FormSchema>;

export function useThreadScreen({
  initialSession,
  initialPage,
  slug,
  thread,
}: Props) {
  const router = useRouter();
  const session = useSession(initialSession);
  const { reportId, resolveReport } = useReportContext();
  const { updateReply, deleteReply, revalidate } = useThreadMutations(
    thread,
    initialPage,
  );

  const [editing, setEditing] = useQueryState("edit", {
    ...parseAsBoolean,
    defaultValue: false,
    clearOnDefault: true,
  });
  const [sortParam, setSortParam] = useQueryState(
    "sort",
    parseAsString.withDefault("desc"),
  );
  const sortOrder: "asc" | "desc" = sortParam === "asc" ? "asc" : "desc";

  const [resetKey, setResetKey] = useState("");
  const [isEmpty, setEmpty] = useState(
    !thread.body || thread.body.trim().length === 0,
  );
  const [attachments, setAttachments] = useState<Asset[]>(() =>
    extractDocumentAssetsFromThread(thread),
  );

  const {
    isConfirming: isConfirmingDelete,
    handleConfirmAction: handleConfirmDelete,
    handleCancelAction: handleCancelDelete,
  } = useConfirmation(handleDeleteThread);

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    reValidateMode: "onChange",
    defaultValues: {
      title: thread.title,
      body: thread.body,
    },
  });

  const { data, error, mutate } = useThreadGet(
    slug,
    {
      page: initialPage?.toString(),
      sort: sortOrder,
    },
    {
      swr: {
        fallbackData: thread,
      },
    },
  );

  // Sync form and attachments when SWR data changes or when entering edit mode
  useEffect(() => {
    if (data) {
      if (!editing) {
        setAttachments(extractDocumentAssetsFromThread(data));
      } else {
        form.reset({
          title: data.title,
          body: data.body,
          tags: data.tags?.map((t) => t.name) ?? [],
        });
      }
    }
  }, [data, editing]);

  const currentBody = form.watch("body");

  // Keep attachments state in sync with editor body removals
  useEffect(() => {
    if (!editing || !thread.body || !currentBody) return;

    // Find what was originally embedded
    const originalEmbeddedPaths = new Set<string>();
    const originalParsed = new DOMParser().parseFromString(thread.body, "text/html");
    originalParsed.querySelectorAll("a, img, span[data-type='file-attachment'], div[data-type='file-attachment']").forEach((el) => {
      const href = el.getAttribute("href") || el.getAttribute("src") || "";
      const path = normalizeAssetPath(href);
      if (path) originalEmbeddedPaths.add(path);
    });

    // Find what is currently embedded
    const currentEmbeddedPaths = new Set<string>();
    const currentParsed = new DOMParser().parseFromString(currentBody, "text/html");
    currentParsed.querySelectorAll("a, img, span[data-type='file-attachment'], div[data-type='file-attachment']").forEach((el) => {
      const href = el.getAttribute("href") || el.getAttribute("src") || "";
      const path = normalizeAssetPath(href);
      if (path) currentEmbeddedPaths.add(path);
    });

    // Remove from attachments state if it was originally embedded but is no longer embedded
    setAttachments((prev) => 
      prev.filter((a) => {
        const path = normalizeAssetPath(a.path ?? a.id);
        if (path && originalEmbeddedPaths.has(path) && !currentEmbeddedPaths.has(path)) {
          return false;
        }
        return true;
      })
    );
  }, [currentBody, editing, thread.body]);

  useBeacon(DatagraphItemKind.thread, data?.id);

  const isModerator = hasPermission(
    session,
    Permission.MANAGE_POSTS,
    Permission.ADMINISTRATOR,
  );

  if (!data) {
    return {
      ready: false as const,
      error,
    };
  }

  function handleEditing() {
    setAttachments(extractDocumentAssetsFromThread(data ?? thread));
    setEditing(true);
  }

  function handleEmptyStateChange(isEmpty: boolean) {
    setEmpty(isEmpty);
  }

  function handleRemoveAsset(assetToRemove: Asset) {
    setAttachments((prev) =>
      prev.filter((a) => {
        if (a.id && assetToRemove.id && a.id === assetToRemove.id) return false;
        const normA = normalizeAssetPath(a.path ?? a.id);
        const normR = normalizeAssetPath(
          assetToRemove.path ?? assetToRemove.id,
        );
        if (normA && normR && normA === normR) return false;
        if (
          a.filename &&
          assetToRemove.filename &&
          a.filename === assetToRemove.filename
        )
          return false;
        return true;
      }),
    );

    const currentBody = form.getValues("body") || "";
    if (currentBody) {
      const parsed = new DOMParser().parseFromString(currentBody, "text/html");
      let modified = false;
      const targetId = assetToRemove.id;
      const targetPath = normalizeAssetPath(
        assetToRemove.path ?? assetToRemove.id,
      );
      const targetFilename = assetToRemove.filename;
      const targetCleanFn = getCleanFilename(targetFilename).toLowerCase();
      const targetNormFn = normalizeFilename(targetFilename);

      parsed
        .querySelectorAll(
          "a, img, span[data-type='file-attachment'], div[data-type='file-attachment']",
        )
        .forEach((el) => {
          const href = el.getAttribute("href") || el.getAttribute("src") || "";
          const fnAttr =
            el.getAttribute("data-filename") ||
            el.getAttribute("download") ||
            el.getAttribute("alt") ||
            "";
          const innerText = el.textContent?.trim() || "";
          const fn = fnAttr || innerText;

          const normHref = normalizeAssetPath(href);
          const elCleanFn = getCleanFilename(fn).toLowerCase();
          const elNormFn = normalizeFilename(fn);

          const matchesId =
            targetId &&
            (href.toLowerCase().includes(targetId.toLowerCase()) ||
              targetId.toLowerCase().includes(href.toLowerCase()));
          const matchesPath =
            targetPath &&
            normHref &&
            (targetPath === normHref ||
              targetPath.includes(normHref) ||
              normHref.includes(targetPath));
          const matchesFilename =
            (targetFilename &&
              fn &&
              (targetFilename.toLowerCase() === fn.toLowerCase() ||
                targetCleanFn === elCleanFn)) ||
            (targetNormFn && elNormFn && targetNormFn === elNormFn);

          if (matchesId || matchesPath || matchesFilename) {
            const parent = el.parentElement;
            el.remove();
            if (
              parent &&
              parent.tagName.toLowerCase() === "p" &&
              parent.innerHTML.trim() === ""
            ) {
              parent.remove();
            }
            modified = true;
          }
        });

      if (modified) {
        form.setValue("body", parsed.body.innerHTML, { shouldDirty: true });
      }
    }

    setResetKey(Date.now().toString());
  }

  function handleDiscardChanges() {
    // TODO: useConfirmation
    form.reset({
      title: thread.title,
      body: thread.body,
      tags: thread.tags.map((t) => t.name),
    });
    setAttachments(extractDocumentAssetsFromThread(data ?? thread));
    setEditing(false);
    setResetKey(Date.now().toString());
    setEmpty(!thread.body || thread.body.trim().length === 0);
  }

  const handleSave = form.handleSubmit(async (formData) => {
    await handle(
      async () => {
        const currentBody = formData.body || "";
        const currentEmbeddedPaths = new Set<string>();
        const parsed = new DOMParser().parseFromString(currentBody, "text/html");
        parsed
          .querySelectorAll(
            "a, img, span[data-type='file-attachment'], div[data-type='file-attachment']",
          )
          .forEach((el) => {
            const href = el.getAttribute("href") || el.getAttribute("src") || "";
            const normP = normalizeAssetPath(href);
            if (normP) {
              currentEmbeddedPaths.add(normP);
            }
            const match = href.match(/\/assets\/([a-zA-Z0-9_-]+)/);
            if (match && match[1]) {
              currentEmbeddedPaths.add(match[1].toLowerCase());
            }
          });

        const originalBody = (data?.body ?? thread.body) || "";
        const originalEmbeddedPaths = new Set<string>();
        const originalParsed = new DOMParser().parseFromString(originalBody, "text/html");
        originalParsed
          .querySelectorAll(
            "a, img, span[data-type='file-attachment'], div[data-type='file-attachment']",
          )
          .forEach((el) => {
            const href = el.getAttribute("href") || el.getAttribute("src") || "";
            const normP = normalizeAssetPath(href);
            if (normP) {
              originalEmbeddedPaths.add(normP);
            }
            const match = href.match(/\/assets\/([a-zA-Z0-9_-]+)/);
            if (match && match[1]) {
              originalEmbeddedPaths.add(match[1].toLowerCase());
            }
          });

        const finalAssetIds = new Set<string>();

        attachments.forEach((a) => {
          if (!a.id) return;
          const normA = normalizeAssetPath(a.path ?? a.id);
          const rawId = !a.id.startsWith("/") ? a.id : null;
          const isCurrentlyEmbedded =
            (normA && currentEmbeddedPaths.has(normA)) ||
            (rawId && currentEmbeddedPaths.has(rawId.toLowerCase()));
          const wasOriginallyEmbedded =
            (normA && originalEmbeddedPaths.has(normA)) ||
            (rawId && originalEmbeddedPaths.has(rawId.toLowerCase()));

          if (isCurrentlyEmbedded || !wasOriginallyEmbedded) {
            if (rawId) {
              finalAssetIds.add(rawId);
            }
          }
        });

        currentEmbeddedPaths.forEach((idOrPath) => {
          if (idOrPath && !idOrPath.includes("/") && idOrPath.length >= 10) {
            finalAssetIds.add(idOrPath);
          }
        });

        const assetIds = Array.from(finalAssetIds);

        await mutate(
          {
            ...(data ?? thread),
            title: formData.title,
            body: formData.body,
            assets: attachments, // this is optimistic local state
          },
          { revalidate: false },
        );

        await threadUpdate(slug, {
          title: formData.title,
          body: formData.body,
          tags: formData.tags,
          asset_ids: assetIds,
        });

        setEditing(false);
        form.reset(formData);
      },
      {
        promiseToast: {
          loading: "Saving...",
          success: "Saved!",
        },
        cleanup: async () => {
          await mutate();
          router.refresh();
        },
      },
    );
  });

  async function handleAcceptThread() {
    await handle(
      async () => {
        await updateReply(thread.id, { visibility: "published" });
        await resolveReport();
      },
      {
        promiseToast: {
          loading: "Accepting...",
          success: "Thread accepted!",
        },
        cleanup: async () => {
          await revalidate();
        },
      },
    );
  }

  function handleEditAndAccept() {
    setAttachments(extractDocumentAssetsFromThread(data ?? thread));
    setEditing(true);
  }

  async function handleDeleteThread() {
    await handle(
      async () => {
        await withUndo({
          message: "Thread deleted",
          duration: 5000,
          toastId: `thread-${thread.id}`,
          action: async () => {
            await deleteReply(thread.id);
            await resolveReport();
            if (reportId) {
              router.push(`/reports`);
            } else {
              router.push("/");
            }
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
    ready: true as const,
    isEditing: editing,
    isEmpty,
    resetKey,
    attachments,
    form,
    isModerator,
    isConfirmingDelete,
    sortOrder,
    data: {
      thread: data,
    },
    handlers: {
      handleEditing,
      handleEmptyStateChange,
      handleRemoveAsset,
      handleDiscardChanges,
      handleSave,
      handleAcceptThread,
      handleEditAndAccept,
      handleConfirmDelete,
      handleCancelDelete,
      handleSetSortOrder: (order: "asc" | "desc") =>
        setSortParam(order === "desc" ? null : order),
    },
  };
}


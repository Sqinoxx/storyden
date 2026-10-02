import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { threadCreate, threadUpdate } from "@/api/openapi-client/threads";
import {
  Asset,
  Thread,
  ThreadInitialProps,
  Visibility,
} from "@/api/openapi-schema";

import { handle } from "@/api/client";
import { NO_CATEGORY_VALUE } from "@/components/category/CategoryTreeSelect/useCategoryTreeSelect";
import { Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import {
  parseTermKey,
  termFor,
  termKey,
  threadTerm,
  writeThreadSemesterMeta,
} from "@/lib/thread/semester";
import { hasPermission } from "@/utils/permissions";

import { normalizeAssetPath } from "@/utils/asset";
import { useTranslation } from "@/lib/i18n";

export type Props = { editing?: string; initialDraft?: Thread };

export const FormShapeSchema = z.object({
  title: z.string().default(""),
  body: z.string().min(1),
  category: z.string().optional(),
  semester: z.string().optional(),
  tags: z.string().array().optional(),
  url: z.string().optional(),
});
export type FormShape = z.infer<typeof FormShapeSchema>;

export function useComposeForm({ initialDraft, editing }: Props) {
  const t = useTranslation();
  const router = useRouter();
  const session = useSession();
  const canPostUncategorised = hasPermission(
    session,
    Permission.POST_IN_ANY_CATEGORY,
  );

  const [isPublishing, setIsPublishing] = useState(false);
  const [isSavingDraft, setIsSavingDraft] = useState(false);
  const [pendingUploads, setPendingUploads] = useState(0);
  const isUploading = pendingUploads > 0;
  const [attachments, setAttachmentsState] = useState<Asset[]>(
    initialDraft?.assets ?? [],
  );

  // The rich editor calls back into handleAttach from long-lived closures and
  // several uploads can finish back to back, so the list is read from a ref
  // rather than whichever render's state the caller happened to capture.
  const attachmentsRef = useRef(attachments);
  const setAttachments = (next: Asset[]) => {
    attachmentsRef.current = next;
    setAttachmentsState(next);
  };

  const draftIdRef = useRef(editing);
  const saveQueueRef = useRef<Promise<unknown>>(Promise.resolve());

  const form = useForm<FormShape>({
    resolver: zodResolver(FormShapeSchema),
    reValidateMode: "onChange",
    defaultValues: initialDraft
      ? {
          title: initialDraft.title,
          body: initialDraft.body,
          tags: initialDraft.tags.map((t) => t.name),
          url: initialDraft.link?.url,
          semester: termKey(threadTerm(initialDraft)),
        }
      : { semester: termKey(termFor(new Date())) },
  });

  const semesterMeta = (raw: string | undefined) => {
    const term = parseTermKey(raw);

    return term ? writeThreadSemesterMeta(initialDraft?.meta, term) : undefined;
  };

  // Serialised so concurrent uploads on a brand new post don't each create
  // their own draft thread.
  const saveDraft = (data: FormShape, overrideAttachments?: Asset[]) => {
    const run = saveQueueRef.current.then(() =>
      saveDraftNow(data, overrideAttachments),
    );
    saveQueueRef.current = run.catch(() => {});
    return run;
  };

  const saveDraftNow = async (
    data: FormShape,
    overrideAttachments?: Asset[],
  ) => {
    const activeAttachments = overrideAttachments ?? attachmentsRef.current;
    const { semester, ...rest } = data;
    const payload: ThreadInitialProps = {
      ...rest,

      // When saving a new draft, these are optional but must be explicitly set.
      title: data.title ?? "",
      body: data.body ?? "",
      url: data.url ?? "",
      tags: data.tags ?? [],
      category: data.category === NO_CATEGORY_VALUE ? undefined : data.category,

      asset_ids: activeAttachments.map((a) => a.id),

      meta: semesterMeta(semester),

      visibility: Visibility.draft,
    };

    if (draftIdRef.current) {
      await threadUpdate(draftIdRef.current, payload);
    } else {
      const { id } = await threadCreate(payload);
      draftIdRef.current = id;

      // Not router.push: that remounts the composer, which aborts any upload
      // still in flight.
      window.history.replaceState(null, "", `/new?id=${id}`);
    }
  };

  const publish = async ({
    title,
    body,
    category,
    semester,
    tags,
    url,
  }: FormShape) => {
    if (title.length < 1) {
      form.setError("title", {
        message: "Your post must have a title to be published",
      });
      return;
    }

    if (!canPostUncategorised && (!category || category === NO_CATEGORY_VALUE)) {
      form.setError("category", {
        message: "You must select a category before publishing",
      });
      return;
    }

    await saveQueueRef.current;

    const draftId = draftIdRef.current;
    if (draftId) {
      const { slug } = await threadUpdate(draftId, {
        title,
        body,
        category: category === NO_CATEGORY_VALUE ? undefined : category,
        visibility: Visibility.published,
        tags,
        url,
        asset_ids: attachmentsRef.current.map((a) => a.id),
        meta: semesterMeta(semester),
      });
      router.push(`/t/${slug}`);
    } else {
      const { slug } = await threadCreate({
        title,
        body,
        category: category === NO_CATEGORY_VALUE ? undefined : category,
        visibility: Visibility.published,
        tags,
        url,
        asset_ids: attachmentsRef.current.map((a) => a.id),
        meta: semesterMeta(semester),
      });
      router.push(`/t/${slug}`);
    }
  };

  const handleSaveDraft = form.handleSubmit((data) =>
    handle(
      async () => {
        setIsSavingDraft(true);
        await saveDraft(data);
      },
      {
        promiseToast: {
          loading: t.toasts.savingDraft,
          success: t.toasts.draftSaved,
        },
        cleanup: async () => {
          setIsSavingDraft(false);
        },
      },
    ),
  );

  const handlePublish = form.handleSubmit((data) =>
    handle(
      async () => {
        setIsPublishing(true);
        await publish(data);
      },
      {
        promiseToast: {
          loading: t.toasts.publishingPost,
          success: t.toasts.postPublished,
        },
        cleanup: async () => {
          setIsPublishing(false);
        },
      },
    ),
  );

  const handleAssetUpload = async (nextAttachments?: Asset[]) => {
    await handle(
      async () => {
        setIsSavingDraft(true);
        const state = form.getValues();
        await saveDraft(state, nextAttachments);
      },
      {
        promiseToast: {
          loading: t.toasts.savingDraft,
          success: t.toasts.draftSaved,
        },
        cleanup: async () => {
          setIsSavingDraft(false);
        },
      },
    );
  };

  const handleAttach = async (a: Asset) => {
    const normNewPath = normalizeAssetPath(a.path ?? a.id);
    const isAlreadyAttached = attachmentsRef.current.some((existing) => {
      if (existing.id && a.id && existing.id === a.id) return true;
      const existingNormPath = normalizeAssetPath(
        existing.path ?? existing.id
      );
      return Boolean(
        normNewPath && existingNormPath && normNewPath === existingNormPath
      );
    });
    if (isAlreadyAttached) return;
    const next = [...attachmentsRef.current, a];
    setAttachments(next);
    await handleAssetUpload(next);
  };

  const handleDetach = async (a: Asset) => {
    const next = attachmentsRef.current.filter((x) => x.id !== a.id);
    setAttachments(next);

    const currentBody = form.getValues("body") || "";
    if (currentBody) {
      const parsed = new DOMParser().parseFromString(currentBody, "text/html");
      let modified = false;
      const targetId = a.id;
      const targetPath = normalizeAssetPath(a.path ?? a.id);
      const targetFilename = a.filename;

      parsed.querySelectorAll("a, img").forEach((el) => {
        const href = el.getAttribute("href") || el.getAttribute("src") || "";
        const fn = el.getAttribute("data-filename") || el.getAttribute("alt") || "";
        const normHref = normalizeAssetPath(href);

        const matchesId = targetId && href.includes(targetId);
        const matchesPath = targetPath && normHref && targetPath === normHref;
        const matchesFilename = targetFilename && fn === targetFilename;

        if (matchesId || matchesPath || matchesFilename) {
          const parent = el.parentElement;
          el.remove();
          if (parent && parent.tagName.toLowerCase() === "p" && parent.innerHTML.trim() === "") {
            parent.remove();
          }
          modified = true;
        }
      });

      if (modified) {
        form.setValue("body", parsed.body.innerHTML);
      }
    }

    await handleAssetUpload(next);
  };

  const handleUploadingChange = (uploading: boolean) => {
    setPendingUploads((n) => Math.max(0, n + (uploading ? 1 : -1)));
  };

  function handleBack() {
    router.back();
  }

  return {
    form,
    state: {
      isPublishing,
      isSavingDraft,
      isUploading,
      attachments,
    },
    handlers: {
      handleSaveDraft,
      handlePublish,
      handleAssetUpload,
      handleAttach,
      handleDetach,
      handleUploadingChange,
      handleBack,
    },
  };
}
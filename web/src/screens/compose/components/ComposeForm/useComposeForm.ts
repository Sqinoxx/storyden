import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
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
import { NO_CATEGORY_VALUE } from "@/components/category/CategorySelect/useCategorySelect";
import { Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { hasPermission } from "@/utils/permissions";

import { normalizeAssetPath } from "@/utils/asset";

export type Props = { editing?: string; initialDraft?: Thread };

export const FormShapeSchema = z.object({
  title: z.string().default(""),
  body: z.string().min(1),
  category: z.string().optional(),
  tags: z.string().array().optional(),
  url: z.string().optional(),
});
export type FormShape = z.infer<typeof FormShapeSchema>;

export function useComposeForm({ initialDraft, editing }: Props) {
  const router = useRouter();
  const session = useSession();
  const canPostUncategorised = hasPermission(
    session,
    Permission.MANAGE_CATEGORIES,
  );

  const [isPublishing, setIsPublishing] = useState(false);
  const [isSavingDraft, setIsSavingDraft] = useState(false);
  const [attachments, setAttachments] = useState<Asset[]>(
  initialDraft?.assets ?? [],
);

  const form = useForm<FormShape>({
    resolver: zodResolver(FormShapeSchema),
    reValidateMode: "onChange",
    defaultValues: initialDraft
      ? {
          title: initialDraft.title,
          body: initialDraft.body,
          tags: initialDraft.tags.map((t) => t.name),
          url: initialDraft.link?.url,
        }
      : {},
  });

  const saveDraft = async (data: FormShape, overrideAttachments?: Asset[]) => {
    const activeAttachments = overrideAttachments ?? attachments;
    const payload: ThreadInitialProps = {
      ...data,

      // When saving a new draft, these are optional but must be explicitly set.
      title: data.title ?? "",
      body: data.body ?? "",
      url: data.url ?? "",
      tags: data.tags ?? [],
      category: data.category === NO_CATEGORY_VALUE ? undefined : data.category,

      asset_ids: activeAttachments.map((a) => a.id),

      visibility: Visibility.draft,
    };

    if (editing) {
      await threadUpdate(editing, payload);
    } else {
      const { id } = await threadCreate(payload);
      router.push(`/new?id=${id}`);
    }
  };

  const publish = async ({ title, body, category, tags, url }: FormShape) => {
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

    if (editing) {
      const { slug } = await threadUpdate(editing, {
        title,
        body,
        category: category === NO_CATEGORY_VALUE ? undefined : category,
        visibility: Visibility.published,
        tags,
        url,
        asset_ids: attachments.map((a) => a.id),
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
        asset_ids: attachments.map((a) => a.id),
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
          loading: "Saving draft...",
          success: "Draft saved!",
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
          loading: "Publishing post...",
          success: "Post published!",
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
          loading: "Saving draft...",
          success: "Draft saved!",
        },
        cleanup: async () => {
          setIsSavingDraft(false);
        },
      },
    );
  };

  const handleAttach = async (a: Asset) => {
    const normNewPath = normalizeAssetPath(a.path ?? a.id);
    const isAlreadyAttached = attachments.some((existing) => {
      if (existing.id && a.id && existing.id === a.id) return true;
      const existingNormPath = normalizeAssetPath(
        existing.path ?? existing.id
      );
      return Boolean(
        normNewPath && existingNormPath && normNewPath === existingNormPath
      );
    });
    if (isAlreadyAttached) return;
    const next = [...attachments, a];
    setAttachments(next);
    await handleAssetUpload(next);
  };

  const handleDetach = async (a: Asset) => {
    const next = attachments.filter((x) => x.id !== a.id);
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

  function handleBack() {
    router.back();
  }

  return {
    form,
    state: {
      isPublishing,
      isSavingDraft,
      attachments,
    },
    handlers: {
      handleSaveDraft,
      handlePublish,
      handleAssetUpload,
      handleAttach,
      handleDetach,
      handleBack,
    },
  };
}
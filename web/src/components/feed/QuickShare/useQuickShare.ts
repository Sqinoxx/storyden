import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { linkCreate } from "@/api/openapi-client/links";
import { Account, Asset, Category, LinkReference } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { NO_CATEGORY_VALUE } from "@/components/category/CategorySelect/useCategorySelect";
import { useFeedMutations } from "@/lib/feed/mutation";
import { useClickAway } from "@/utils/useClickAway";

import { normalizeAssetPath } from "@/utils/asset";

export type Props = {
  initialSession?: Account;
  initialCategory?: Category | null;
  showCategorySelect: boolean;
};

export const FormSchema = z.object({
  title: z.string().optional(),
  body: z.string().optional(),
  category: z.string().optional(),
  tags: z.string().array().optional(),
});
export type Form = z.infer<typeof FormSchema>;

export function useQuickShare({ initialCategory }: Props) {
  const session = useSession();
  const [editing, setEditing] = useState(false);
  const [postURL, setPostURL] = useState<string | null>(null);
  const [hydratedLink, setHydratedLink] = useState<
    LinkReference | "loading" | null
  >(null);
  const formRef = useClickAway<HTMLFormElement>(() => setEditing(false));
  const [resetKey, setResetKey] = useState("");
  const [uploadedAssets, setUploadedAssets] = useState<Asset[]>([]);

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      title: "",
      body: undefined,
      category: initialCategory?.id,
    },
  });

  // Watch body for changes - because onChange is already used by RHF.
  const bodyContent = form.watch("body");

  // When the body changes, find the first URL in the content and remember it.
  useEffect(() => {
    const parsed = new DOMParser().parseFromString(
      bodyContent || "",
      "text/html",
    );
    const url = getFirstURL(parsed);

    if (url) {
      setPostURL(url);
    } else {
      setPostURL(null);
      setHydratedLink(null);
    }
  }, [bodyContent]);

  // When the URL found in the body changes, index/fetch its rich preview.
  useEffect(() => {
    if (postURL) {
      setHydratedLink("loading");
      linkCreate({ url: postURL }).then((link) => {
        // Only hydrate link if there's a preview available. Otherwise bail out.
        if (link.title && link.description) {
          setHydratedLink(link);
        } else {
          setHydratedLink(null);
        }
      });
    }
  }, [postURL]);

  const { createThread, revalidate } = useFeedMutations(session, {
    categories:
      initialCategory === undefined
        ? undefined
        : initialCategory === null
          ? ["null"]
          : [initialCategory.slug],
  });

  const handleAddAssets = (assets: Asset[]) => {
    setUploadedAssets((prev) => {
      const existingPaths = new Set(
        prev.map((a) => normalizeAssetPath(a.path ?? a.id))
      );
      const filtered = assets.filter((a) => {
        const normP = normalizeAssetPath(a.path ?? a.id);
        return !normP || !existingPaths.has(normP);
      });
      return [...prev, ...filtered];
    });
  };

  const handleRemoveAsset = (assetId: string) => {
    const targetAsset = uploadedAssets.find((a) => a.id === assetId || a.filename === assetId);
    setUploadedAssets((prev) => prev.filter((a) => a.id !== assetId && a.filename !== assetId));

    const currentBody = form.getValues("body") || "";
    if (currentBody) {
      const parsed = new DOMParser().parseFromString(currentBody, "text/html");
      let modified = false;

      const normTargetPath = targetAsset ? normalizeAssetPath(targetAsset.path ?? targetAsset.id) : null;
      const targetFilename = targetAsset?.filename ?? assetId;

      parsed.querySelectorAll("a, img").forEach((el) => {
        const href = el.getAttribute("href") || el.getAttribute("src") || "";
        const fn = el.getAttribute("data-filename") || el.getAttribute("alt") || "";
        const normHrefPath = normalizeAssetPath(href);

        const matchesId = assetId && href.includes(assetId);
        const matchesPath = normTargetPath && normHrefPath && normTargetPath === normHrefPath;
        const matchesFn = targetFilename && fn === targetFilename;

        if (matchesId || matchesPath || matchesFn) {
          const parent = el.parentElement;
          el.remove();
          if (parent && parent.tagName.toLowerCase() === "p" && parent.innerHTML.trim() === "") {
            parent.remove();
          }
          modified = true;
        }
      });

      if (modified) {
        const newHtml = parsed.body.innerHTML;
        form.setValue("body", newHtml);
        setResetKey(new Date().toISOString());
      }
    }
  };

  // Awaited so react-hook-form keeps isSubmitting true for the whole request.
  // Without it the submit button re-enables mid-flight and a second click posts
  // the thread twice.
  const handlePost = form.handleSubmit(async (data: Form) => {
    await handle(
      async () => {
        const titleInput = data.title?.trim();
        const parsed = new DOMParser().parseFromString(
          bodyContent || "",
          "text/html",
        );

        let title = "";
        let body = "";
        let isFallback = false;

        if (titleInput && titleInput.length > 0) {
          title = titleInput;
          body = bodyContent || "";
        } else if (bodyContent && bodyContent.trim().length > 0) {
          const split = splitTitleBody(parsed);
          title = split.title;
          body = split.body;
          isFallback = split.isFallback;
        }

        const linkAvailable =
          hydratedLink && hydratedLink !== "loading" ? hydratedLink : undefined;

        const threadTitle = isFallback
          ? linkAvailable?.title
            ? linkAvailable.title
            : title
          : title;

        if (!threadTitle) {
          throw new Error("Cannot post an empty thread.");
        }

        // Filter uploadedAssets to only include assets that are actually still referenced in body HTML
        const activeUploadedAssets = uploadedAssets.filter((a) => {
          const targetId = a.id;
          const normP = normalizeAssetPath(a.path ?? a.id);
          const fn = a.filename;
          return Array.from(parsed.querySelectorAll("a, img")).some((el) => {
            const href = el.getAttribute("href") || el.getAttribute("src") || "";
            const elFn = el.getAttribute("data-filename") || el.getAttribute("alt") || "";
            const normHref = normalizeAssetPath(href);
            return (
              (targetId && href.includes(targetId)) ||
              (normP && normHref && normP === normHref) ||
              (fn && elFn === fn)
            );
          });
        });

        const assetIds = Array.from(
          new Set(activeUploadedAssets.map((a) => a.id).filter(Boolean)),
        );

        const newThread = {
          title: threadTitle,
          body,
          url: postURL ?? undefined,
          category:
            data.category === NO_CATEGORY_VALUE ? undefined : data.category,
          tags: data.tags && data.tags.length > 0 ? data.tags : undefined,
          visibility: "published" as const,
          asset_ids: assetIds.length > 0 ? assetIds : undefined,
        };

        await createThread(newThread, linkAvailable);

        // Reset the rich text editor and title input
        setResetKey(new Date().toISOString());

        form.resetField("title");
        form.resetField("body");
        form.setValue("tags", []);
        setUploadedAssets([]);

        setHydratedLink(null);
      },
      {
        async cleanup() {
          await revalidate();
        },
      },
    );
  });

  function handleFocus() {
    setEditing(true);
  }

  return {
    form,
    state: {
      formRef,
      editing,
      hydratedLink,
      resetKey,
      uploadedAssets,
    },
    handlers: {
      handleFocus,
      handlePost,
      handleAddAssets,
      handleRemoveAsset,
    },
  };
}

function getFirstURL(html: Document) {
  const result = html.querySelector("a");
  if (!result) {
    return undefined;
  }

  const href = result?.attributes.getNamedItem("href")?.value;
  if (!href?.startsWith("https")) {
    return undefined;
  }

  return href;
}

function splitTitleBody(html: Document) {
  const bodyEl = html.querySelector("body");

  // The title is the first text node of the first paragraph element. If the
  // paragraph contains any more tags after the initial text node, do not include
  // them in the title, this ensures that <a> tags are not included.
  const firstChild = bodyEl?.querySelector("p")?.childNodes[0] as Node;

  if (!firstChild) {
    throw new Error("Not enough text content to post a new thread.");
  }

  let isFallback = false;

  const title = ((): string => {
    const textContent = firstChild.textContent;

    switch (firstChild.nodeType) {
      case Node.ELEMENT_NODE:
        if (firstChild.nodeName === "A") {
          const el = firstChild as HTMLAnchorElement;

          // File attachment nodes: use the filename as the title and keep
          // the attachment in the body (do not treat it like a regular link).
          if (el.getAttribute("data-type") === "file-attachment") {
            const fileName =
              el.getAttribute("data-filename") ||
              el.textContent?.trim() ||
              "Attachment";
            // isFallback stays false so the filename is always used as title.
            return fileName;
          }

          // Mark this as a fallback strategy, if the Link acquired in the
          // actual link fetch yields a title from the opengraph data, and this
          // is true, then we'll use the opengraph title instead.
          isFallback = true;

          if (textContent?.startsWith("http")) {
            // If the anchor tag is just a bare tag (where the text content is
            // the actual URL itself, no title), then we'll just use that.
            const parsed = new URL(textContent);
            return parsed.hostname;
          } else if (textContent === "") {
            // Otherwise, if there's no text content in the node, try to get the
            // href attribute and use the hostname from that as the title.
            const href = (firstChild as any) /* Types are broken here */
              ?.getAttribute("href");

            if (href) {
              const parsed = new URL(href);
              return parsed.hostname;
            }

            return textContent ?? "";
          } else {
            // Finally, if none of the above conditions are met, just use the
            // text content of the node. Which might be empty, but caught below.
            return textContent ?? "";
          }
        }
    }

    return textContent ?? "";
  })();

  // We want something of substance to post a new thread. This may bug out tho.
  if (!title) {
    throw new Error("Not enough text content to post a new thread.");
  }

  // For file attachments, keep the entire body intact (don't strip the first
  // child node) so the attachment link remains downloadable in the post.
  const firstChildEl =
    firstChild.nodeType === Node.ELEMENT_NODE
      ? (firstChild as HTMLElement)
      : null;
  const isFileAttachment =
    firstChildEl?.getAttribute("data-type") === "file-attachment";

  if (!isFileAttachment) {
    // Remove the first text node from the first paragraph so it doesn't
    // appear duplicated in the body (it becomes the thread title above).
    bodyEl?.querySelector("p")?.childNodes[0]?.remove();
  }

  const body = bodyEl?.getHTML() ?? "";

  return {
    title,
    body,
    isFallback,
  };
}

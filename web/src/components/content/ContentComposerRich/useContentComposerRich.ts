"use client";

import { FocusClasses } from "@tiptap/extension-focus";
import { Link } from "@tiptap/extension-link";
import Placeholder from "@tiptap/extension-placeholder";
import { generateHTML, generateJSON } from "@tiptap/html";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { EditorView } from "@tiptap/pm/view";
import { Extension, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { ChangeEvent, useEffect, useId, useRef, useState } from "react";

import { Asset } from "@/api/openapi-schema";

import { handle } from "@/api/client";
import { css } from "@/styled-system/css";
import { getAssetURL } from "@/utils/asset";

import { ContentComposerProps } from "../composer-props";
import {
  hasAssetFile,
  isSupportedAsset,
  useImageUpload,
} from "../useImageUpload";

import { ImageExtended, uploadPositionsKey } from "./plugins/ImagePlugin";
import {
  FileAttachmentExtended,
  isDocumentHref,
} from "./plugins/FileAttachmentPlugin";
import { LinkPasteMenuPlugin } from "./plugins/LinkPasteMenuPlugin";
import { LinkPreview } from "./plugins/LinkPreviewPlugin";

const ERROR_UNSUPPORTED_FILE_TYPE = "File type not supported";

export type Block = "p" | "h1" | "h2" | "h3" | "h4" | "h5" | "h6";

export function useContentComposer(props: ContentComposerProps) {
  const { uploadWithProgress } = useImageUpload();
  const [uploadingCount, setUploadingCount] = useState(0);
  const [isDragging, setIsDragging] = useState(false);
  const [isDragError, setIsDragError] = useState(false);
  const [dragErrorMessage, setDragErrorMessage] = useState("");
  const [dragFileCount, setDragFileCount] = useState(0);
  const dragCounterRef = useRef(0);
  const uploadCounterRef = useRef(0);
  const activeUploadsRef = useRef<
    Map<
      string,
      {
        abortController: AbortController;
        blobUrl: string;
        file: File;
        status: "uploading" | "failed" | "completed";
      }
    >
  >(new Map());

  // Extension to detect and abort uploads when image nodes are deleted
  const UploadCleanupExtension = Extension.create({
    name: "uploadCleanup",

    addProseMirrorPlugins() {
      return [
        new Plugin({
          key: new PluginKey("uploadCleanup"),
          appendTransaction(_transactions, oldState, newState) {
            const oldUploadingIds = new Set<string>();
            oldState.doc.descendants((node) => {
              if ((node.type.name === "image" || node.type.name === "fileAttachment") && node.attrs["data-upload-id"]) {
                oldUploadingIds.add(node.attrs["data-upload-id"]);
              }
            });

            const newUploadingIds = new Set<string>();
            newState.doc.descendants((node) => {
              if ((node.type.name === "image" || node.type.name === "fileAttachment") && node.attrs["data-upload-id"]) {
                newUploadingIds.add(node.attrs["data-upload-id"]);
              }
            });

            const deletedUploadIds = Array.from(oldUploadingIds).filter(
              (id) => !newUploadingIds.has(id),
            );

            deletedUploadIds.forEach((uploadId) => {
              const upload = activeUploadsRef.current.get(uploadId);
              if (upload) {
                upload.abortController.abort("embed deleted");
                URL.revokeObjectURL(upload.blobUrl);
                console.debug(
                  `Upload for ID: ${uploadId} aborted due to node deletion.`,
                );
                activeUploadsRef.current.delete(uploadId);
              }
            });

            return null;
          },
        }),
      ];
    },
  });

  const extensions = [
    StarterKit,
    FocusClasses,
    LinkPreview,
    LinkPasteMenuPlugin,
    Link.configure({
      // Disable navigation when clicking links in the editor if it's active.
      openOnClick: props.disabled ? true : false,
    }).extend({
      inclusive: false,
      parseHTML() {
        return [
          {
            tag: "a[href]:not([data-display]):not([data-type='file-attachment'])",
            getAttrs: (dom) => {
              const href = dom.getAttribute("href");
              if (!href || isDocumentHref(href)) {
                return false;
              }
              return { href };
            },
          },
        ];
      },
    }),

    ImageExtended.configure({
      allowBase64: false,
      HTMLAttributes: {
        class: css({ borderRadius: "md" }),
      },
      handleFiles,
      handleRetry,
      handleCancel,
    }),
    FileAttachmentExtended.configure({
      handleFiles,
      handleRetry,
      handleCancel,
    }),
    Placeholder.configure({
      placeholder: props.placeholder ?? "Write your heart out...",
      includeChildren: true,
      showOnlyCurrent: false,
    }),
    UploadCleanupExtension,
  ];

  // This is for the initial server render.
  const initialValueJSON = generateJSON(
    props.initialValue ?? "<p></p>",
    extensions,
  );
  const initialValueHTML = generateHTML(initialValueJSON, extensions);

  // Each editor needs a unique ID for the menu's file upload input ID.
  const uniqueID = useId();

  const editor = useEditor({
    immediatelyRender: false,
    editorProps: {
      attributes: {
        "data-editor-id": uniqueID,
        class: css({
          height: "full",
          width: "full",
        }),
      },
    },
    extensions,
    content: props.initialValue ?? "<p></p>",
    onUpdate: ({ editor }) => {
      let html = editor.getHTML();

      // Filter out elements that are still uploading or failed
      const parser = new DOMParser();
      const doc = parser.parseFromString(html, "text/html");
      const uploadingElements = doc.querySelectorAll(
        '[data-uploading="true"]',
      );
      const failedElements = doc.querySelectorAll("[data-upload-error]");

      uploadingElements.forEach((el) => el.remove());
      failedElements.forEach((el) => el.remove());

      html = doc.body.innerHTML;

      props.onChange?.(html, editor.isEmpty);
    },
  });

  const prevResetKeyRef = useRef(props.resetKey);

  useEffect(() => {
    if (!editor) {
      return;
    }

    const resetKeyChanged = prevResetKeyRef.current !== props.resetKey;
    prevResetKeyRef.current = props.resetKey;

    if (resetKeyChanged) {
      queueMicrotask(() => {
        if (editor && !editor.isDestroyed) {
          editor.commands.setContent(props.value ?? props.initialValue ?? "<p></p>");
        }
      });
      return;
    }

    if (props.value !== undefined && props.value !== editor.getHTML()) {
      queueMicrotask(() => {
        if (editor && !editor.isDestroyed && props.value !== undefined && props.value !== editor.getHTML()) {
          editor.commands.setContent(props.value);
        }
      });
    }
  }, [editor, props.initialValue, props.value, props.resetKey]);

  useEffect(() => {
    if (!editor) {
      return;
    }

    editor.setEditable(!props.disabled, false);
  }, [editor, props.disabled]);

  // Navigating away mid-upload otherwise leaves the XHRs running against a
  // composer that no longer exists, and leaks every placeholder's object URL for
  // the lifetime of the document. Uploads that already completed are left alone
  // — their blob URL has been swapped for the asset URL and revoking is handled
  // at that point.
  useEffect(() => {
    const uploads = activeUploadsRef.current;

    return () => {
      uploads.forEach((upload) => {
        if (upload.status === "uploading") {
          upload.abortController.abort("composer unmounted");
        }
        URL.revokeObjectURL(upload.blobUrl);
      });
      uploads.clear();
    };
  }, []);

  // -
  // Image uploading logic.
  // -

  async function handleProgress(
    view: EditorView,
    uploadId: string,
    percent: number,
  ) {
    if (!view) {
      throw new Error("Unable to access text editor state.");
    }

    const positionMap = uploadPositionsKey.getState(view.state);
    const pos = positionMap?.get(uploadId);

    if (pos === undefined) {
      console.warn(
        `handleProgress: No position found for upload ID: ${uploadId}`,
      );
      return;
    }

    const node = view.state.doc.nodeAt(pos);
    if (node) {
      const tr = view.state.tr.setNodeMarkup(pos, undefined, {
        ...node.attrs,
        "data-upload-progress": Math.round(percent).toString(),
      });
      view.dispatch(tr);
    }
  }

  function markUploadAsFailed(view: EditorView, uploadId: string) {
    const currentState = view.state;

    const positionMap = uploadPositionsKey.getState(view.state);
    const pos = positionMap?.get(uploadId);

    if (pos === undefined) {
      console.warn(
        `markUploadAsFailed: No position found for upload ID: ${uploadId}`,
      );
      return;
    }

    const errorTransaction = currentState.tr.setNodeMarkup(pos, undefined, {
      ...currentState.doc.nodeAt(pos)?.attrs,
      "data-uploading": null,
      "data-upload-error": "Upload failed",
    });

    view.dispatch(errorTransaction);

    const upload = activeUploadsRef.current.get(uploadId);
    if (upload) {
      upload.status = "failed";
    }
  }

  function handleRetry(view: EditorView, uploadId: string) {
    if (!view) {
      throw new Error("Unable to access text editor state.");
    }

    const upload = activeUploadsRef.current.get(uploadId);
    if (!upload) {
      console.warn(`No active upload found for ID: ${uploadId}`);
      return;
    }

    const positionMap = uploadPositionsKey.getState(view.state);
    const pos = positionMap?.get(uploadId);

    if (pos === undefined) {
      console.warn(`handleRetry: No position found for upload ID: ${uploadId}`);
      return;
    }

    const abortController = new AbortController();

    activeUploadsRef.current.set(uploadId, {
      ...upload,
      abortController,
      status: "uploading",
    });

    const uploadingTransaction = view.state.tr.setNodeMarkup(pos, undefined, {
      ...view.state.doc.nodeAt(pos)?.attrs,
      "data-uploading": "true",
      "data-upload-error": null,
    });
    view.dispatch(uploadingTransaction);

    setUploadingCount((prev) => prev + 1);

    // NOTE: No await here, we allow the handler to not block as we update the
    // the button inside the image node via the upload progress itself.
    handle(
      async () => {
        const asset = await uploadWithProgress(
          upload.file,
          (percent) => handleProgress(view, uploadId, percent),
          undefined,
          abortController,
        );

        // we search for the node again because the position might have changed.

        const positionMap = uploadPositionsKey.getState(view.state);
        const pos = positionMap?.get(uploadId);

        if (pos === undefined) {
          console.warn(
            `handleRetry finished: No position found for upload ID: ${uploadId}`,
          );
          return;
        }

        const isPdfNode = view.state.doc.nodeAt(pos)?.type.name === "fileAttachment";
        const updateAttrs = isPdfNode
          ? {
            href: getAssetURL(asset.path),
            fileName: upload.file.name,
            "data-upload-id": null,
            "data-uploading": null,
            "data-upload-error": null,
            "data-upload-progress": null,
          }
          : {
            src: getAssetURL(asset.path),
            alt: upload.file.name,
            "data-upload-id": null,
            "data-uploading": null,
            "data-upload-error": null,
            "data-upload-progress": null,
          };

        const updateTransaction = view.state.tr.setNodeMarkup(pos, undefined, updateAttrs);

        view.dispatch(updateTransaction);

        URL.revokeObjectURL(upload.blobUrl);
        const trackedUpload = activeUploadsRef.current.get(uploadId);
        if (trackedUpload) {
          trackedUpload.status = "completed";
        }

        props.onAssetUpload?.(asset);
      },
      {
        onError: async () => {
          markUploadAsFailed(view, uploadId);
        },
        cleanup: async () => {
          setUploadingCount((prev) => Math.max(0, prev - 1));
        },
      },
    );
  }

  function handleCancel(view: EditorView, uploadId: string) {
    if (!view) {
      throw new Error("Unable to access text editor state.");
    }

    const upload = activeUploadsRef.current.get(uploadId);

    if (!upload) {
      console.warn(`No active upload found for ID: ${uploadId}`);
      return;
    }

    const positionMap = uploadPositionsKey.getState(view.state);
    const pos = positionMap?.get(uploadId);

    if (pos === undefined) {
      console.warn(
        `handleCancel: No position found for upload ID: ${uploadId}`,
      );
      return;
    }

    console.debug(
      `Cancelling upload for ID: ${uploadId} and removing node at position ${pos}`,
    );

    const nodeSize = view.state.doc.nodeAt(pos)?.nodeSize ?? 1;
    const transaction = view.state.tr.delete(pos, pos + nodeSize);
    view.dispatch(transaction);

    URL.revokeObjectURL(upload.blobUrl);
    activeUploadsRef.current.delete(uploadId);
  }

  async function handleFiles(view: EditorView, files: File[]) {
    if (!view) {
      throw new Error("Unable to access text editor state.");
    }

    const { state } = view;
    const { selection } = state;
    const { schema } = view.state;
    const imageNode = schema.nodes?.["image"];
    const fileAttachmentNode = schema.nodes?.["fileAttachment"];

    if (!imageNode || !fileAttachmentNode) {
      return [];
    }

    const assets: Asset[] = [];

    // Advances past each placeholder as it is inserted. Reusing the original
    // offset for every file would stack them all at the same point, so a
    // multi-file drop would come out in reverse order.
    let insertPos = selection.$head.pos;

    for (const f of files) {
      uploadCounterRef.current += 1;
      const uploadId = `upload-${Date.now()}-${uploadCounterRef.current}`;
      const isAttachment = !f.type.startsWith("image/");

      if (isAttachment) {
        // Non-image files (PDFs, docs, ZIPs) are attached at the bottom only, not inserted into the editor content.
        const abortController = new AbortController();
        setUploadingCount((prev) => prev + 1);

        handle(
          async () => {
            const asset = await uploadWithProgress(
              f,
              () => {},
              undefined,
              abortController,
            );
            const assetWithFilename = { ...asset, filename: f.name };
            assets.push(assetWithFilename);
            props.onAssetUpload?.(assetWithFilename);
          },
          {
            onError: async (err) => {
              console.error("Attachment upload failed:", err);
            },
            cleanup: async () => {
              setUploadingCount((prev) => prev - 1);
            },
          },
        );
        continue;
      }

      const nodeType = imageNode;

      // Create blob URL for immediate preview
      const blobUrl = URL.createObjectURL(f);

      // Create abort controller for this upload
      const abortController = new AbortController();

      // Track this upload
      activeUploadsRef.current.set(uploadId, {
        abortController,
        blobUrl,
        file: f,
        status: "uploading",
      });

      // Insert placeholder immediately
      const attrs = {
        src: blobUrl,
        alt: f.name,
        "data-upload-id": uploadId,
        "data-uploading": "true",
      };

      const placeholderNode = nodeType.create(attrs);

      const insertTransaction = view.state.tr.insert(
        insertPos,
        placeholderNode,
      );
      view.dispatch(insertTransaction);

      insertPos += placeholderNode.nodeSize;

      setUploadingCount((prev) => prev + 1);

      handle(
        async () => {
          const asset = await uploadWithProgress(
            f,
            (percent) => handleProgress(view, uploadId, percent),
            undefined,
            abortController,
          );

          // Find the node with this upload-id and update it
          const currentState = view.state;
          let nodePos: number | null = null;

          currentState.doc.descendants((node, pos) => {
            if (
              node.type.name === "image" &&
              node.attrs["data-upload-id"] === uploadId
            ) {
              nodePos = pos;
              return false; // Stop searching
            }
            return true; // Continue searching
          });

          if (nodePos !== null) {
            const newAttrs = {
              src: getAssetURL(asset.path),
              alt: f.name,
              "data-upload-id": null,
              "data-uploading": null,
              "data-upload-error": null,
              "data-upload-progress": null,
            };
            // Update the node with the real URL and remove upload attrs
            const updateTransaction = currentState.tr.setNodeMarkup(
              nodePos,
              undefined,
              newAttrs,
            );

            view.dispatch(updateTransaction);

            // Clean up blob URL and mark as completed
            URL.revokeObjectURL(blobUrl);
            const upload = activeUploadsRef.current.get(uploadId);
            if (upload) {
              upload.status = "completed";
            }

            const assetWithFilename = { ...asset, filename: f.name };
            assets.push(assetWithFilename);
            props.onAssetUpload?.(assetWithFilename);
          }
        },
        {
          onError: async () => {
            markUploadAsFailed(view, uploadId);
          },
          cleanup: async () => {
            setUploadingCount((prev) => prev - 1);
          },
        },
      );
    }

    return assets;
  }

  async function handleFileUpload(e: ChangeEvent<HTMLInputElement>) {
    if (!e.currentTarget.files || !editor) {
      return;
    }

    const files = Array.from(e.currentTarget.files).filter((file) =>
      isSupportedAsset(file.type),
    );

    await handleFiles(editor.view, files);
    e.target.value = "";
  }

  // -
  // Text formatting logic.
  // -

  function handleBlockType(kind: Block) {
    switch (kind) {
      case "p":
        editor?.chain().focus().setParagraph().run();
        break;
      case "h1":
        editor?.chain().focus().setHeading({ level: 1 }).run();
        break;
      case "h2":
        editor?.chain().focus().setHeading({ level: 2 }).run();
        break;

      case "h3":
        editor?.chain().focus().setHeading({ level: 3 }).run();
        break;

      case "h4":
        editor?.chain().focus().setHeading({ level: 4 }).run();
        break;

      case "h5":
        editor?.chain().focus().setHeading({ level: 5 }).run();
        break;

      case "h6":
        editor?.chain().focus().setHeading({ level: 6 }).run();
        break;
    }
  }

  function getBlockType(): Block | null {
    if (editor?.isActive("paragraph")) return "p";
    if (editor?.isActive("heading", { level: 1 })) return "h1";
    if (editor?.isActive("heading", { level: 2 })) return "h2";
    if (editor?.isActive("heading", { level: 3 })) return "h3";
    if (editor?.isActive("heading", { level: 4 })) return "h4";
    if (editor?.isActive("heading", { level: 5 })) return "h5";
    if (editor?.isActive("heading", { level: 6 })) return "h6";

    return "p";
  }

  function getDragOverlayMessage() {
    if (isDragError) {
      return dragErrorMessage;
    }
    return dragFileCount === 1
      ? "Drop 1 file to upload"
      : `Drop ${dragFileCount} files to upload`;
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault();
  }

  function handleDragEnter(e: React.DragEvent) {
    // Incremented unconditionally so it stays paired with every dragleave,
    // including the ones fired by dragging existing content (e.g. an image)
    // around within the editor — those carry text/html, not a file, but the
    // browser still fires enter/leave for them. Gating the increment on
    // hasFile while the decrement below stays unconditional would drift the
    // counter negative, and the drop overlay would ultimately depend on this
    // value never reliably reaching zero again for the rest of the session.
    dragCounterRef.current += 1;

    const items = Array.from(e.dataTransfer.items);
    const hasFile = items.some((item) => item.kind === "file");

    if (!hasFile) {
      return;
    }

    e.preventDefault();
    setIsDragging(true);

    const hasAsset = hasAssetFile(e.dataTransfer.items);
    const assetCount = items.filter((item) =>
      isSupportedAsset(item.type),
    ).length;

    if (!hasAsset) {
      setIsDragError(true);
      setDragErrorMessage(ERROR_UNSUPPORTED_FILE_TYPE);
    } else {
      setIsDragError(false);
      setDragErrorMessage("");
    }

    setDragFileCount(assetCount);
  }

  function handleDragLeave() {
    dragCounterRef.current = Math.max(0, dragCounterRef.current - 1);
    if (dragCounterRef.current === 0) {
      setIsDragging(false);
      setIsDragError(false);
      setDragErrorMessage("");
      setDragFileCount(0);
    }
  }

  async function handleDrop(e: React.DragEvent) {
    e.preventDefault();

    dragCounterRef.current = 0;
    setIsDragging(false);

    if (!editor) {
      throw new Error("Unable to access text editor state.");
    }

    const files = Array.from(e.dataTransfer.files);
    const assets = files.filter((file) => isSupportedAsset(file.type));

    if (assets.length > 0) {
      setIsDragError(false);
      setDragErrorMessage("");
      setDragFileCount(0);
      await handleFiles(editor.view, assets);
      return;
    }

    // A non-file drop (repositioning content within the editor, dropping a
    // link/text) legitimately has no files — only surface an error when the
    // drag was actually carrying files but none survived to the drop event,
    // otherwise this would flag normal in-editor drag-and-drop as broken.
    const wasFileDrag = Array.from(e.dataTransfer.types).includes("Files");
    if (!wasFileDrag) {
      return;
    }

    setIsDragError(true);
    setDragErrorMessage(
      files.length > 0 ? ERROR_UNSUPPORTED_FILE_TYPE : "Drop not recognised, please try again",
    );
    setDragFileCount(0);
    setIsDragging(true);
    setTimeout(() => {
      setIsDragging(false);
      setIsDragError(false);
      setDragErrorMessage("");
    }, 2000);
  }

  return {
    editor,
    uniqueID,
    initialValueHTML,
    uploadingCount,
    isDragging,
    isDragError,
    getDragOverlayMessage,
    handlers: {
      handleFileUpload,
      handleDragOver,
      handleDragEnter,
      handleDragLeave,
      handleDrop,
    },
    format: {
      text: {
        active: getBlockType(),
        set: handleBlockType,
      },

      // Marks
      bold: {
        isActive: editor?.isActive("bold") ?? false,
        isDisabled: editor?.can().toggleBold() === false,
        toggle: () => editor?.chain().focus().toggleBold().run(),
      },
      italic: {
        isActive: editor?.isActive("italic") ?? false,
        isDisabled: editor?.can().toggleItalic() === false,
        toggle: () => editor?.chain().focus().toggleItalic().run(),
      },
      strike: {
        isActive: editor?.isActive("strike") ?? false,
        isDisabled: editor?.can().toggleStrike() === false,
        toggle: () => editor?.chain().focus().toggleStrike().run(),
      },
      code: {
        isActive: editor?.isActive("code") ?? false,
        isDisabled: editor?.can().toggleCode() === false,
        toggle: () => editor?.chain().focus().toggleCode().run(),
      },

      // Blocks
      blockquote: {
        isActive: editor?.isActive("blockquote") ?? false,
        isDisabled: editor?.can().toggleBlockquote() === false,
        toggle: () => editor?.chain().focus().toggleBlockquote().run(),
      },
      pre: {
        isActive: editor?.isActive("codeBlock") ?? false,
        isDisabled: editor?.can().toggleCodeBlock() === false,
        toggle: () => editor?.chain().focus().toggleCodeBlock().run(),
      },
      bulletList: {
        isActive: editor?.isActive("bulletList") ?? false,
        isDisabled: editor?.can().toggleBulletList() === false,
        toggle: () => editor?.chain().focus().toggleBulletList().run(),
      },
      orderedList: {
        isActive: editor?.isActive("orderedList") ?? false,
        isDisabled: editor?.can().toggleOrderedList() === false,
        toggle: () => editor?.chain().focus().toggleOrderedList().run(),
      },
    },
  };
}

import {
  Node,
  mergeAttributes,
} from "@tiptap/core";
import {
  NodeViewProps,
  NodeViewWrapper,
  ReactNodeViewRenderer,
} from "@tiptap/react";
import { Plugin } from "prosemirror-state";
import { EditorView } from "prosemirror-view";
import { useState, useCallback } from "react";
import React from "react";

import { Asset } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { ProgressCircle } from "@/components/ui/progress";
import { Download, Eye, Trash2 } from "lucide-react";
import { css } from "@/styled-system/css";
import { styled } from "@/styled-system/jsx";
import { getCleanFilename } from "@/utils/asset";
import { FilePreviewModal, isPreviewableAsset } from "@/components/post/FilePreviewModal";

export interface FileAttachmentOptions {
  handleFiles: (view: EditorView, files: File[]) => Promise<Asset[]>;
  handleRetry: (view: EditorView, uploadId: string) => void;
  handleCancel: (view: EditorView, uploadId: string) => void;
}

function Component(props: NodeViewProps) {
  const isUploading = props.node.attrs["data-uploading"] === "true";
  const uploadError = props.node.attrs["data-upload-error"];
  const uploadId = props.node.attrs["data-upload-id"];
  const uploadProgress = props.node.attrs["data-upload-progress"];
  const progressPercent = uploadProgress ? parseInt(uploadProgress, 10) : 0;

  const href = props.node.attrs["href"];
  const rawFileName = props.node.attrs["fileName"] || "Attachment";
  const fileName = getCleanFilename(rawFileName);

  const [isEditable, setIsEditable] = useState(props.editor.isEditable);
  const isSelected = props.selected && isEditable;

  React.useEffect(() => {
    const updateEditable = () => {
      setIsEditable(props.editor.isEditable);
    };
    props.editor.on("transaction", updateEditable);
    return () => {
      props.editor.off("transaction", updateEditable);
    };
  }, [props.editor]);

  const { handleRetry, handleCancel } =
    props.extension.options as FileAttachmentOptions;

  // Preview state
  const canPreview = !isUploading && isPreviewableAsset(undefined, fileName);
  const [previewOpen, setPreviewOpen] = useState(false);

  const handleDeleteNode = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      props.deleteNode();
    },
    [props]
  );

  const handleDownload = useCallback(async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (isUploading) { e.preventDefault(); return; }
    if (href) {
      e.preventDefault();
      try {
        const res = await fetch(href);
        if (!res.ok) throw new Error("Download failed");
        const blob = await res.blob();
        const blobUrl = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = blobUrl;
        link.download = fileName;
        document.body.appendChild(link);
        link.click();
        link.remove();
        URL.revokeObjectURL(blobUrl);
      } catch {
        const link = document.createElement("a");
        link.href = href;
        link.download = fileName;
        document.body.appendChild(link);
        link.click();
        link.remove();
      }
    }
  }, [href, fileName, isUploading]);

  const handlePreview = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setPreviewOpen(true);
  }, []);

  const handleBadgeClick = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (canPreview) {
      setPreviewOpen(true);
    } else {
      handleDownload(e);
    }
  }, [canPreview, handleDownload]);

  return (
    <>
      <NodeViewWrapper
        as="span"
        className={css({
          display: "inline-block",
          marginLeft: "1",
          marginRight: "1",
          verticalAlign: "middle",
        })}
      >
        <styled.div
          display="inline-flex"
          alignItems="center"
          gap="2"
          padding="2"
          paddingRight="3"
          borderRadius="md"
          borderWidth="thin"
          borderColor="border.default"
          backgroundColor={isSelected ? "bg.muted" : "bg.default"}
          color="fg.default"
          position="relative"
          overflow="hidden"
          onClick={!isUploading && href ? handleBadgeClick : undefined}
          title={!isUploading && href ? (canPreview ? `${fileName} – Vorschau öffnen` : `${fileName} – Herunterladen`) : fileName}
          style={{ cursor: !isUploading && href ? "pointer" : "default" }}
          _hover={{ backgroundColor: "bg.muted" }}
        >
          {isUploading ? (
            <styled.span
              display="flex"
              alignItems="center"
              justifyContent="center"
              backgroundColor="bg.muted"
              borderRadius="sm"
              padding="1"
            >
              <ProgressCircle value={progressPercent} size="sm" />
            </styled.span>
          ) : (
            <styled.span display="flex" alignItems="center" flexShrink="0" fontSize="sm">
              📄
            </styled.span>
          )}

          <styled.span fontSize="sm" fontWeight="medium" truncate maxWidth="xs">
            {fileName}
          </styled.span>

          {!isUploading && (
            <styled.span display="inline-flex" alignItems="center" gap="1.5" color="fg.muted" ml="1">
              {/* Preview eye */}
              {href && canPreview && (
                <styled.button
                  type="button"
                  onClick={handlePreview}
                  title="Vorschau"
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                  color="fg.muted"
                  _hover={{ color: "fg.default" }}
                  style={{
                    cursor: "pointer",
                    background: "transparent",
                    border: "none",
                    padding: "2px",
                  }}
                >
                  <Eye size={14} />
                </styled.button>
              )}
              {/* Download */}
              {href && (
                <styled.button
                  type="button"
                  onClick={handleDownload}
                  title="Herunterladen"
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                  color="fg.muted"
                  _hover={{ color: "fg.default" }}
                  style={{
                    cursor: "pointer",
                    background: "transparent",
                    border: "none",
                    padding: "2px",
                  }}
                >
                  <Download size={14} />
                </styled.button>
              )}
              {/* Remove attachment when editing */}
              {isEditable && (
                <styled.button
                  type="button"
                  onClick={handleDeleteNode}
                  title="Datei entfernen"
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                  color="fg.muted"
                  _hover={{ color: "red.5" }}
                  style={{
                    cursor: "pointer",
                    background: "transparent",
                    border: "none",
                    padding: "2px",
                    marginLeft: "2px",
                  }}
                >
                  <Trash2 size={14} />
                </styled.button>
              )}
            </styled.span>
          )}

          {uploadError && (
            <styled.div
              position="absolute"
              inset="0"
              display="flex"
              alignItems="center"
              justifyContent="center"
              backgroundColor="bg.error"
              color="fg.error"
              gap="2"
            >
              <styled.span fontSize="xs">Failed</styled.span>
              <Button
                type="button"
                size="xs"
                variant="outline"
                onClick={(e) => {
                  e.preventDefault();
                  handleRetry(props.view, uploadId);
                }}
              >
                Retry
              </Button>
              <Button
                type="button"
                size="xs"
                variant="ghost"
                onClick={(e) => {
                  e.preventDefault();
                  handleCancel(props.view, uploadId);
                }}
              >
                Remove
              </Button>
            </styled.div>
          )}
        </styled.div>
      </NodeViewWrapper>

      {/* Preview modal */}
      {previewOpen && href && (
        <FilePreviewModal
          url={href}
          displayName={fileName}
          onClose={() => setPreviewOpen(false)}
          onDownload={() => handleDownload({ preventDefault: () => {}, stopPropagation: () => {} } as React.MouseEvent)}
        />
      )}
    </>
  );
}

export function isDocumentHref(href: string | null): boolean {
  if (!href) return false;
  const lower = href.toLowerCase();
  const hasDocExt = /\.(pdf|docx?|xlsx?|pptx?|zip|rar|7z|txt|csv)(\?.*)?$/i.test(lower);
  const isAssetUrl = lower.includes("/api/assets/") || lower.includes("/assets/");
  const isImage = /\.(png|jpe?g|gif|webp|svg|bmp)(\?.*)?$/i.test(lower);
  return (hasDocExt || isAssetUrl) && !isImage;
}

export const FileAttachmentExtended = Node.create<FileAttachmentOptions>({
  name: "fileAttachment",
  group: "inline",
  inline: true,
  atom: true,
  selectable: true,

  addAttributes() {
    return {
      href: {
        default: null,
        parseHTML: (element) => element.getAttribute("href"),
        renderHTML: (attributes) => {
          if (!attributes["href"]) return {};
          return { href: attributes["href"] };
        },
      },
      fileName: {
        default: null,
        parseHTML: (element) =>
          element.getAttribute("data-filename") || element.textContent?.trim() || null,
        renderHTML: (attributes) => {
          if (!attributes["fileName"]) return {};
          return { "data-filename": attributes["fileName"] };
        },
      },
      "data-upload-id": {
        default: null,
        parseHTML: (element) => element.getAttribute("data-upload-id"),
        renderHTML: (attributes) => {
          if (!attributes["data-upload-id"]) return {};
          return { "data-upload-id": attributes["data-upload-id"] };
        },
      },
      "data-uploading": {
        default: null,
        parseHTML: (element) => element.getAttribute("data-uploading"),
        renderHTML: (attributes) => {
          if (!attributes["data-uploading"]) return {};
          return { "data-uploading": attributes["data-uploading"] };
        },
      },
      "data-upload-error": {
        default: null,
        parseHTML: (element) => element.getAttribute("data-upload-error"),
        renderHTML: (attributes) => {
          if (!attributes["data-upload-error"]) return {};
          return { "data-upload-error": attributes["data-upload-error"] };
        },
      },
      "data-upload-progress": {
        default: null,
        parseHTML: (element) => element.getAttribute("data-upload-progress"),
        renderHTML: (attributes) => {
          if (!attributes["data-upload-progress"]) return {};
          return { "data-upload-progress": attributes["data-upload-progress"] };
        },
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: 'a[data-type="file-attachment"]',
      },
      {
        tag: 'a[href]',
        getAttrs: (dom) => {
          const element = dom as HTMLElement;
          const href = element.getAttribute("href");
          if (isDocumentHref(href)) {
            return {
              href,
              fileName:
                element.getAttribute("data-filename") ||
                element.textContent?.trim() ||
                "Attachment",
            };
          }
          return false;
        },
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    const fileName =
      HTMLAttributes["data-filename"] || HTMLAttributes["fileName"] || "Attachment";
    return [
      "a",
      mergeAttributes(HTMLAttributes, {
        "data-type": "file-attachment",
        target: "_blank",
        download: fileName,
      }),
      fileName,
    ];
  },

  addNodeView() {
    return ReactNodeViewRenderer(Component);
  },

  addProseMirrorPlugins() {
    const handleFiles = this.options.handleFiles;
    return [
      new Plugin({
        props: {
          handlePaste(view, event) {
            if (!event.clipboardData) {
              return false;
            }

            const files: File[] = [];

            if (event.clipboardData.items?.length) {
              for (const item of event.clipboardData.items) {
                if (item.kind === "file") {
                  const file = item.getAsFile();
                  if (file) {
                    files.push(file);
                  }
                }
              }
            }

            // Let Image plugin handle images
            const nonImages = files.filter((file) => !/image/i.test(file.type));

            // Right now, only intercept PDFs
            const pdfs = nonImages.filter((f) => f.type === "application/pdf");

            if (pdfs.length === 0) {
              return false;
            }

            event.preventDefault();
            handleFiles(view, pdfs);
            return true;
          },
        },
      }),
    ];
  },
});

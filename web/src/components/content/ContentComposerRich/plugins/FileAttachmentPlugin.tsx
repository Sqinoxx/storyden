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

import { Asset } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { ProgressCircle } from "@/components/ui/progress";
import { FileIcon } from "lucide-react";
import { css } from "@/styled-system/css";
import { styled } from "@/styled-system/jsx";

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
  const fileName = props.node.attrs["fileName"] || "Attachment";

  const isEditable = props.editor.isEditable;
  const isSelected = props.selected && isEditable;

  const { handleRetry, handleCancel } =
    props.extension.options as FileAttachmentOptions;

  return (
    <NodeViewWrapper
      as="span"
      className={css({
        display: "inline-block",
        marginLeft: "1",
        marginRight: "1",
        verticalAlign: "middle",
      })}
    >
      <styled.a
        href={isUploading ? undefined : href}
        target="_blank"
        rel="noopener noreferrer"
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
        textDecoration="none"
        position="relative"
        overflow="hidden"
        style={{ cursor: isUploading ? "default" : "pointer" }}
        _hover={!isUploading ? { backgroundColor: "bg.muted" } : undefined}
      >
        <styled.span
          display="flex"
          alignItems="center"
          justifyContent="center"
          backgroundColor="bg.muted"
          borderRadius="sm"
          padding="1"
        >
          {isUploading ? (
            <ProgressCircle value={progressPercent} size="sm" />
          ) : (
            <FileIcon size={16} />
          )}
        </styled.span>

        <styled.span fontSize="sm" fontWeight="medium" truncate maxWidth="xs">
          {fileName}
        </styled.span>

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
      </styled.a>
    </NodeViewWrapper>
  );
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
          element.getAttribute("data-filename") || element.textContent || null,
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
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return [
      "a",
      mergeAttributes(HTMLAttributes, {
        "data-type": "file-attachment",
        target: "_blank",
      }),
      HTMLAttributes["data-filename"] || HTMLAttributes["fileName"] || "Attachment",
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

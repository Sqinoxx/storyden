import { Editor } from "@tiptap/react";
import { PaperclipIcon, XIcon } from "lucide-react";
import { useEffect } from "react";

import { isFileLink } from "@/components/drive/resolveDroppedDriveFile";
import { Spinner } from "@/components/ui/Spinner";
import { Button } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { CardIcon } from "@/components/ui/icons/Card";
import { LinkIcon } from "@/components/ui/icons/Link";
import { useTranslation } from "@/lib/i18n";
import { HStack, styled } from "@/styled-system/jsx";

import { linkPasteMenuKey } from "./plugins/LinkPasteMenuPlugin";
import { PastedFileLink } from "./useContentComposerRich";

const SETTLED_VISIBLE_MS = 15000;

type Props = {
  editor: Editor;
  pastedFileLink: PastedFileLink | null;
  onReplaceFileLink: (kind: "link" | "preview") => void;
  onDismissFileLink: () => void;
  onUploadAsFile?: (url: string, range: { from: number; to: number }) => void;
};

export function LinkPasteMenu({
  editor,
  pastedFileLink,
  onReplaceFileLink,
  onDismissFileLink,
  onUploadAsFile,
}: Props) {
  const t = useTranslation();

  const settled = pastedFileLink && pastedFileLink.status !== "loading";
  useEffect(() => {
    if (!settled) return;
    const timer = setTimeout(onDismissFileLink, SETTLED_VISIBLE_MS);
    return () => clearTimeout(timer);
  }, [settled, pastedFileLink?.id, onDismissFileLink]);

  if (pastedFileLink) {
    const canReplace = pastedFileLink.status !== "attached";

    return (
      <Bar label={t.link.pastedFileLink}>
        <HStack gap="1.5" flexGrow="1" flexShrink="0" color="fg.default">
          {pastedFileLink.status === "loading" ? (
            <Spinner size="sm" />
          ) : (
            <PaperclipIcon width="14" height="14" />
          )}
          <styled.span fontWeight="medium" whiteSpace="nowrap">
            {pastedFileLink.status === "loading"
              ? t.link.fileLoading
              : pastedFileLink.status === "attached"
                ? t.link.fileAttached
                : t.link.fileInserted}
          </styled.span>
        </HStack>

        {canReplace && (
          <HStack gap="1" flexWrap="wrap">
            <styled.span color="fg.muted">{t.link.instead}</styled.span>
            <Button
              type="button"
              size="xs"
              variant="outline"
              onClick={() => onReplaceFileLink("link")}
            >
              <LinkIcon /> {t.link.asLink}
            </Button>
            <Button
              type="button"
              size="xs"
              variant="outline"
              onClick={() => onReplaceFileLink("preview")}
            >
              <CardIcon /> {t.link.preview}
            </Button>
          </HStack>
        )}

        <DismissButton label={t.link.dismiss} onClick={onDismissFileLink} />
      </Bar>
    );
  }

  const menuState = linkPasteMenuKey.getState(editor.state);
  if (!menuState?.isVisible || !menuState.url || !menuState.range) {
    return null;
  }

  const { url, range } = menuState;
  const canUploadAsFile = !!onUploadAsFile && isFileLink(url);

  const dismissMenu = () => {
    editor.view.dispatch(
      editor.state.tr.setMeta(linkPasteMenuKey, {
        isVisible: false,
        url: null,
        position: 0,
        range: null,
      }),
    );
  };

  const handleLinkChoice = () => {
    editor
      .chain()
      .focus()
      .setTextSelection(range)
      .setLink({ href: url })
      .setTextSelection(range.to)
      .run();

    dismissMenu();
  };

  const handleCardChoice = () => {
    editor
      .chain()
      .focus()
      .deleteRange(range)
      .setLinkPreview({ href: url })
      .run();

    dismissMenu();
  };

  const handleFileChoice = () => {
    dismissMenu();
    onUploadAsFile?.(url, range);
  };

  return (
    <Bar label={t.link.showLinkAs}>
      <styled.span color="fg.muted">{t.link.showLinkAs}</styled.span>
      <HStack gap="1" flexWrap="wrap">
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={handleLinkChoice}
        >
          <LinkIcon /> {t.link.text}
        </Button>
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={handleCardChoice}
        >
          <CardIcon /> {t.link.preview}
        </Button>
        {canUploadAsFile && (
          <Button
            type="button"
            size="xs"
            variant="outline"
            onClick={handleFileChoice}
          >
            <PaperclipIcon /> {t.link.file}
          </Button>
        )}
      </HStack>
      <DismissButton label={t.link.dismiss} onClick={dismissMenu} />
    </Bar>
  );
}

function Bar({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <HStack
      role="group"
      aria-label={label}
      data-link-paste-bar
      position="sticky"
      bottom="0"
      zIndex="docked"
      w="full"
      minW="0"
      flexWrap="wrap"
      gap="2"
      px="2"
      py="1.5"
      borderRadius="md"
      borderWidth="thin"
      borderColor="border.subtle"
      bg="bg.muted"
      boxShadow="xs"
      fontSize="xs"
      // Clicking a choice must not blur the editor first: on touch devices the
      // keyboard closing reflows the page and the tap lands somewhere else.
      onMouseDown={(e) => e.preventDefault()}
    >
      {children}
    </HStack>
  );
}

function DismissButton({
  label,
  onClick,
}: {
  label: string;
  onClick: () => void;
}) {
  return (
    <IconButton
      type="button"
      size="xs"
      variant="ghost"
      ml="auto"
      aria-label={label}
      title={label}
      onClick={onClick}
    >
      <XIcon />
    </IconButton>
  );
}

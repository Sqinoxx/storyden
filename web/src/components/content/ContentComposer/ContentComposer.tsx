"use client";

import { useSession } from "@/auth";
import { CardBox } from "@/styled-system/jsx";
import { center } from "@/styled-system/patterns";

import { ContentComposerMarkdown } from "../ContentComposerMarkdown/ContentComposerMarkdown";
import { ContentComposerRich } from "../ContentComposerRich/ContentComposerRich";
import { ContentComposerProps } from "../composer-props";

export function ContentComposer(props: ContentComposerProps) {
  const session = useSession();

  // Always use rich-text renderer in read-only/disabled mode so HTML content
  // (including file attachment cards) is displayed correctly regardless of
  // the user's preferred editor mode.
  if (props.disabled) {
    return <ContentComposerRich {...props} />;
  }

  const editorMode = session?.meta.editor.mode;

  switch (editorMode) {
    case "richtext":
      return <ContentComposerRich {...props} />;

    case "markdown":
      return <ContentComposerMarkdown {...props} />;

    default:
      return <ContentComposerRich {...props} />;
  }
}

import { Plugin, PluginKey } from "@tiptap/pm/state";
import { Extension } from "@tiptap/react";
import { normalizeLink } from "@/lib/link/validation";

export type LinkPasteMenuState = {
  isVisible: boolean;
  url: string | null;
  position: number;
  range: { from: number; to: number } | null;
};

export const linkPasteMenuKey = new PluginKey<LinkPasteMenuState>(
  "linkPasteMenu",
);

export type LinkPasteMenuOptions = {
  // Returning true takes over the pasted link; the menu is not shown.
  onPasteLink?: (url: string, range: { from: number; to: number }) => boolean;
};

export const LinkPasteMenuPlugin = Extension.create<LinkPasteMenuOptions>({
  name: "linkPasteMenu",

  addOptions() {
    return { onPasteLink: undefined };
  },

  addProseMirrorPlugins() {
    const options = this.options;

    return [
      new Plugin<LinkPasteMenuState>({
        key: linkPasteMenuKey,

        state: {
          init() {
            return {
              isVisible: false,
              url: null,
              position: 0,
              range: null,
            };
          },

          apply(tr, value) {
            const meta = tr.getMeta(linkPasteMenuKey);

            if (meta) {
              return meta;
            }

            if (!tr.docChanged) {
              return value;
            }

            if (value.isVisible) {
              return {
                isVisible: false,
                url: null,
                position: 0,
                range: null,
              };
            }

            return value;
          },
        },

        props: {
          handlePaste(view, event) {
            const { state } = view;
            const { selection } = state;
            const { $from } = selection;

            const parentNode = $from.parent;
            const isEmptyParagraph =
              parentNode.type.name === "paragraph" &&
              parentNode.content.size === 0;

            if (!isEmptyParagraph) {
              return false;
            }

            const clipboardData = event.clipboardData;
            if (!clipboardData) {
              return false;
            }

            const pastedText = clipboardData.getData("text/plain").trim();

            if (!pastedText) {
              return false;
            }

            const url = getSinglePastedUrl(pastedText);
            if (!url) {
              return false;
            }

            event.preventDefault();

            const insertPos = selection.from;
            const tr = state.tr.insertText(pastedText, insertPos);

            const newPos = insertPos + pastedText.length;
            const range = { from: insertPos, to: newPos };

            if (options.onPasteLink) {
              view.dispatch(tr);
              if (options.onPasteLink(url, range)) {
                return true;
              }

              view.dispatch(
                view.state.tr.setMeta(linkPasteMenuKey, {
                  isVisible: true,
                  url,
                  position: newPos,
                  range,
                }),
              );
              return true;
            }

            tr.setMeta(linkPasteMenuKey, {
              isVisible: true,
              url,
              position: newPos,
              range: { from: insertPos, to: newPos },
            });

            view.dispatch(tr);

            return true;
          },
        },
      }),
    ];
  },
});

function getSinglePastedUrl(text: string): string | null {
  if (/\s/.test(text)) {
    return null;
  }

  if (/^https?:\/\//i.test(text)) {
    try {
      return new URL(text).toString();
    } catch {
      return null;
    }
  }

  return normalizeLink(text) ?? null;
}

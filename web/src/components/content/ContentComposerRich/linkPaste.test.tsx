import { fireEvent, render, waitFor, within } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ContentComposerRich } from "./ContentComposerRich";

const assetImport = vi.fn();
vi.mock("@/api/openapi-client/assets", async () => {
  const actual = await vi.importActual<
    typeof import("@/api/openapi-client/assets")
  >("@/api/openapi-client/assets");

  return { ...actual, assetImport: (body: unknown) => assetImport(body) };
});

vi.mock("@tiptap/react", async () => {
  const actual =
    await vi.importActual<typeof import("@tiptap/react")>("@tiptap/react");

  return { ...actual, BubbleMenu: () => null };
});

vi.mock("@/api/openapi-client/links", () => ({
  useLinkCreate: () => ({
    data: undefined,
    error: undefined,
    isMutating: false,
    trigger: vi.fn(),
  }),
}));

let latestHTML = "";

const PDF_URL = "https://register.awmf.org/assets/Kariesprophylaxe.pdf";

const PDF_ASSET = {
  id: "imported",
  filename: "Kariesprophylaxe.pdf",
  path: "/api/assets/imported-kariesprophylaxe-pdf",
  mime_type: "application/pdf",
  width: 0,
  height: 0,
};

function Controlled() {
  const [value, setValue] = useState<string | undefined>(undefined);

  return (
    <ContentComposerRich
      inlineAttachments
      value={value}
      onChange={(next) => {
        latestHTML = next;
        setValue(next);
      }}
    />
  );
}

async function renderEditor() {
  const { container } = render(<Controlled />);

  await waitFor(() => {
    expect(container.querySelector(".ProseMirror")).not.toBeNull();
  });

  const editor = container.querySelector(".ProseMirror") as HTMLElement;
  return { container, editor };
}

function paste(editor: HTMLElement, text: string) {
  fireEvent.paste(editor, {
    clipboardData: {
      types: ["text/plain"],
      getData: (type: string) => (type === "text/plain" ? text : ""),
    },
  });
}

const bar = (c: HTMLElement) =>
  c.querySelector<HTMLElement>("[data-link-paste-bar]");

const choice = (c: HTMLElement, name: string) =>
  within(bar(c)!).getByRole("button", { name });

beforeEach(() => {
  assetImport.mockReset();
  Range.prototype.getClientRects = () => [] as unknown as DOMRectList;
  Range.prototype.getBoundingClientRect = () => new DOMRect();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("pasting links", () => {
  it("uploads a pasted file link straight away and offers the alternatives", async () => {
    assetImport.mockResolvedValue(PDF_ASSET);
    const { container, editor } = await renderEditor();

    paste(editor, PDF_URL);

    await waitFor(() => {
      expect(bar(container)?.textContent).toContain("Als Datei eingefügt");
    });

    expect(assetImport).toHaveBeenCalledWith({ url: PDF_URL });
    expect(container.textContent).toContain("Kariesprophylaxe.pdf");
    expect(editor.querySelector(`a[href="${PDF_URL}"]`)).toBeNull();
    expect(choice(container, "Link")).not.toBeNull();
    expect(choice(container, "Vorschau")).not.toBeNull();
  });

  it("swaps the uploaded file for a link when asked", async () => {
    assetImport.mockResolvedValue(PDF_ASSET);
    const { container, editor } = await renderEditor();

    paste(editor, PDF_URL);

    await waitFor(() => {
      expect(bar(container)?.textContent).toContain("Als Datei eingefügt");
    });

    fireEvent.click(choice(container, "Link"));

    await waitFor(() => {
      expect(editor.querySelector(`a[href="${PDF_URL}"]`)).not.toBeNull();
    });
    expect(latestHTML).not.toContain("file-attachment");
    expect(editor.textContent).toBe(PDF_URL);
    expect(bar(container)).toBeNull();
  });

  it("drops a file that is still loading when a preview is chosen", async () => {
    let finish: (asset: typeof PDF_ASSET) => void = () => {};
    assetImport.mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const { container, editor } = await renderEditor();

    paste(editor, PDF_URL);

    await waitFor(() => {
      expect(bar(container)?.textContent).toContain("Datei wird übernommen");
    });

    fireEvent.click(choice(container, "Vorschau"));
    finish(PDF_ASSET);

    await waitFor(() => {
      expect(latestHTML).toContain('data-display="card"');
    });
    await new Promise((r) => setTimeout(r, 20));
    expect(latestHTML).not.toContain("file-attachment");
  });

  it("keeps asking how to show an ordinary link", async () => {
    const url = "https://www.zm-online.de/news/aktuelles";
    const { container, editor } = await renderEditor();

    paste(editor, url);

    await waitFor(() => {
      expect(bar(container)?.textContent).toContain("Link anzeigen als");
    });
    expect(assetImport).not.toHaveBeenCalled();
    expect(within(bar(container)!).queryByRole("button", { name: "Datei" })).toBeNull();

    fireEvent.click(choice(container, "Text"));

    await waitFor(() => {
      expect(editor.querySelector(`a[href="${url}"]`)).not.toBeNull();
    });
    expect(bar(container)).toBeNull();
  });
});

import { fireEvent, render, waitFor } from "@testing-library/react";
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

vi.mock("../useImageUpload", async () => {
  const actual =
    await vi.importActual<typeof import("../useImageUpload")>(
      "../useImageUpload",
    );

  return {
    ...actual,
    useImageUpload: () => ({
      upload: vi.fn(),
      uploadWithProgress: () => new Promise(() => {}),
    }),
  };
});

const DRIVE_URL = `${window.location.origin}/drive/dbf0dtosi88000bln04g/1wLKOp83T2tUPw5neR2qhioik8UQLX5so`;

function Controlled() {
  const [value, setValue] = useState<string | undefined>(undefined);

  return (
    <ContentComposerRich
      inlineAttachments
      value={value}
      onChange={(next) => setValue(next)}
    />
  );
}

function urlDataTransfer(url: string) {
  const data: Record<string, string> = {
    "text/uri-list": url,
    "text/plain": url,
  };

  return {
    items: [
      { kind: "string", type: "text/uri-list" },
      { kind: "string", type: "text/plain" },
    ],
    types: ["text/uri-list", "text/plain"],
    files: [],
    getData: (type: string) => data[type] ?? "",
  };
}

async function renderEditor() {
  const { container } = render(<Controlled />);

  await waitFor(() => {
    expect(container.querySelector(".ProseMirror")).not.toBeNull();
  });

  const editor = container.querySelector(".ProseMirror") as HTMLElement;
  document.elementFromPoint = () => editor;

  return { container, editor };
}

beforeEach(() => {
  Range.prototype.getClientRects = () => [] as unknown as DOMRectList;
  Range.prototype.getBoundingClientRect = () => new DOMRect();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("ContentComposerRich drive link drops", () => {
  it("uploads a dropped drive link to a pdf instead of inserting the link", async () => {
    const fetchMock = vi.fn((_input: RequestInfo | URL) => {
      return Promise.resolve(
        new Response("%PDF-1.4", {
          headers: {
            "content-type": "application/pdf",
            "content-disposition": `attachment; filename="Aspirin - Fachinformation.pdf"`,
          },
        }),
      );
    });
    vi.stubGlobal("fetch", fetchMock);

    const { container, editor } = await renderEditor();

    fireEvent.drop(editor, { dataTransfer: urlDataTransfer(DRIVE_URL) });

    await waitFor(() => {
      expect(container.textContent).toContain("Aspirin - Fachinformation.pdf");
    });

    expect(editor.textContent).not.toContain("/drive/");
    expect(
      fetchMock.mock.calls.filter(([input]) =>
        String(input).includes("/api/drive/"),
      ),
    ).toHaveLength(1);
  });

  it("keeps the link when the drive entry is not a pdf or word file", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        Promise.resolve(
          new Response("hello", { headers: { "content-type": "text/plain" } }),
        ),
      ),
    );

    const { editor } = await renderEditor();

    fireEvent.drop(editor, { dataTransfer: urlDataTransfer(DRIVE_URL) });

    await waitFor(() => {
      expect(editor.querySelector(`a[href="${DRIVE_URL}"]`)).not.toBeNull();
    });
  });

  it("imports a dropped link to a pdf on another site as an attachment", async () => {
    const url =
      "https://register.awmf.org/assets/guidelines/Kariesprophylaxe.pdf";
    assetImport.mockResolvedValue({
      id: "imported",
      filename: "Kariesprophylaxe.pdf",
      path: "/api/assets/imported-kariesprophylaxe-pdf",
      mime_type: "application/pdf",
      width: 0,
      height: 0,
    });

    const { container, editor } = await renderEditor();

    fireEvent.drop(editor, { dataTransfer: urlDataTransfer(url) });

    await waitFor(() => {
      expect(container.textContent).toContain("Kariesprophylaxe.pdf");
    });

    expect(assetImport).toHaveBeenCalledWith({ url });
    expect(editor.querySelector(`a[href="${url}"]`)).toBeNull();
  });

  it("falls back to the link when the import is refused", async () => {
    const url = "https://example.com/private/Skript.pdf";
    assetImport.mockRejectedValue(new Error("400"));

    const { editor } = await renderEditor();

    fireEvent.drop(editor, { dataTransfer: urlDataTransfer(url) });

    await waitFor(() => {
      expect(editor.querySelector(`a[href="${url}"]`)).not.toBeNull();
    });
  });

  it("leaves ordinary web links to the editor", async () => {
    const url = "https://www.zm-online.de/news/aktuelles";

    const { editor } = await renderEditor();

    fireEvent.drop(editor, { dataTransfer: urlDataTransfer(url) });

    await waitFor(() => {
      expect(editor.textContent).toContain(url);
    });
    expect(assetImport).not.toHaveBeenCalled();
  });
});

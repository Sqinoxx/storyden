import { fireEvent, render, waitFor } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ContentComposerRich } from "./ContentComposerRich";

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
});

import { fireEvent, render, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

// The bubble menu positions itself with tippy, whose default export does not
// survive jsdom's ESM interop - it throws on the first editor transaction,
// which is every assertion here. Nothing below depends on the menu existing.
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
      // Never settles: the assertions are about what the document looks like
      // while an upload is still in flight.
      uploadWithProgress: () => new Promise(() => {}),
    }),
  };
});

import { ContentComposerRich } from "./ContentComposerRich";

// The composer strips in-flight upload placeholders out of the HTML it reports
// through onChange, so a controlled consumer (quick share) always feeds back a
// value that is missing them. Comparing that value against the raw document
// then looks like an external edit and setContent wipes the placeholder — which
// the cleanup plugin reads as a deletion and aborts the upload behind it, so
// dropping a file appears to do nothing at all.
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

function fileDataTransfer(file: File) {
  return {
    items: [{ kind: "file", type: file.type }],
    types: ["Files"],
    files: [file],
  };
}

describe("ContentComposerRich in-flight uploads", () => {
  it("keeps the image placeholder in the document while a controlled value round-trips", async () => {
    const { container } = render(<Controlled />);

    const root = container.querySelector(
      "[id^='rich-text-editor-']",
    ) as HTMLElement;

    await waitFor(() => {
      expect(container.querySelector(".ProseMirror")).not.toBeNull();
    });

    const file = new File(["x"], "screenshot.png", { type: "image/png" });
    fireEvent.drop(root, { dataTransfer: fileDataTransfer(file) });

    await waitFor(() => {
      expect(
        container.querySelector('.ProseMirror img[data-uploading="true"]'),
      ).not.toBeNull();
    });

    // Give the value round-trip a chance to land before asserting it survived.
    await new Promise((resolve) => setTimeout(resolve, 50));

    expect(
      container.querySelector('.ProseMirror img[data-uploading="true"]'),
    ).not.toBeNull();
  });

  it("embeds a dropped document as an attachment node", async () => {
    const { container } = render(<Controlled />);

    const root = container.querySelector(
      "[id^='rich-text-editor-']",
    ) as HTMLElement;

    await waitFor(() => {
      expect(container.querySelector(".ProseMirror")).not.toBeNull();
    });

    const file = new File(["x"], "handout.pdf", { type: "application/pdf" });
    fireEvent.drop(root, { dataTransfer: fileDataTransfer(file) });

    await waitFor(() => {
      expect(container.textContent).toContain("handout.pdf");
    });
  });
});

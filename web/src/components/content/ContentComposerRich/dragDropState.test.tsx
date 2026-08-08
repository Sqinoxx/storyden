import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ContentComposerRich } from "./ContentComposerRich";

// dragenter/dragleave fire for ANY native drag, not just external file
// drags — including repositioning an image within the editor (text/html
// payload) or dragging selected text. handleDragEnter/handleDragLeave use a
// counter to know when the pointer has fully left the editor, and that
// counter must stay balanced across both kinds of drag or the overlay can
// get stuck (or never appear) on the next real file drop.
function dataTransfer(kind: "file" | "string", type: string) {
  return {
    items: [{ kind, type }],
    types: kind === "file" ? ["Files"] : [type],
    files: [],
  };
}

describe("ContentComposerRich drag overlay state", () => {
  it("stays balanced across a non-file drag so a later file drag can still hide the overlay on dragleave", async () => {
    const { container } = render(<ContentComposerRich />);

    const root = container.querySelector(
      "[id^='rich-text-editor-']",
    ) as HTMLElement;

    await waitFor(() => {
      expect(container.querySelector(".ProseMirror")).not.toBeNull();
    });

    // An in-editor drag (e.g. repositioning an existing image) carries
    // text/html, not a file.
    fireEvent.dragEnter(root, { dataTransfer: dataTransfer("string", "text/html") });
    fireEvent.dragLeave(root);

    expect(screen.queryByRole("status")).toBeNull();

    // A real external file drag should now show the overlay...
    fireEvent.dragEnter(root, { dataTransfer: dataTransfer("file", "image/png") });
    expect(screen.getByRole("status")).not.toBeNull();

    // ...and hide it again if the user drags back out without dropping.
    fireEvent.dragLeave(root);
    expect(screen.queryByRole("status")).toBeNull();
  });
});

import { act, fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { ContentComposerRich } from "./ContentComposerRich";

vi.mock("@tiptap/react", async () => {
  const actual =
    await vi.importActual<typeof import("@tiptap/react")>("@tiptap/react");

  return { ...actual, BubbleMenu: () => null };
});

function transfer(types: string[]) {
  return {
    types,
    items: types.map((type) => ({
      kind: type === "Files" ? "file" : "string",
      type: type === "Files" ? "application/pdf" : type,
    })),
    files: [],
    getData: () => "",
  };
}

async function renderEditor(disabled = false) {
  const { container } = render(
    <ContentComposerRich disabled={disabled} inlineAttachments />,
  );

  await waitFor(() => {
    expect(container.querySelector(".ProseMirror")).not.toBeNull();
  });

  return container;
}

const hint = (c: HTMLElement) => c.querySelector('[data-drop-hint="true"]');

beforeEach(() => {
  Range.prototype.getClientRects = () => [] as unknown as DOMRectList;
  Range.prototype.getBoundingClientRect = () => new DOMRect();
});

afterEach(() => {
  fireEvent.drop(window, { dataTransfer: transfer([]) });
  vi.useRealTimers();
});

describe("drop hint", () => {
  it("outlines the editor while a file is dragged over the page", async () => {
    const container = await renderEditor();

    expect(hint(container)).toBeNull();

    fireEvent.dragOver(document.body, { dataTransfer: transfer(["Files"]) });

    await waitFor(() => expect(hint(container)).not.toBeNull());
    expect(hint(container)?.textContent).toBe("Hier ablegen");

    fireEvent.drop(window, { dataTransfer: transfer(["Files"]) });

    await waitFor(() => expect(hint(container)).toBeNull());
  });

  it("outlines the editor for a link dragged from another window", async () => {
    const container = await renderEditor();

    fireEvent.dragEnter(document.body, {
      dataTransfer: transfer(["DownloadURL", "text/plain"]),
    });

    await waitFor(() => expect(hint(container)).not.toBeNull());
  });

  it("ignores drags that start on the page itself", async () => {
    const container = await renderEditor();

    fireEvent.dragStart(document.body, {
      dataTransfer: transfer(["text/uri-list"]),
    });
    fireEvent.dragOver(document.body, {
      dataTransfer: transfer(["text/uri-list"]),
    });

    expect(hint(container)).toBeNull();
    fireEvent.dragEnd(document.body);
  });

  it("ignores plain text drags", async () => {
    const container = await renderEditor();

    fireEvent.dragOver(document.body, {
      dataTransfer: transfer(["text/plain", "text/html"]),
    });

    expect(hint(container)).toBeNull();
  });

  it("does not outline read-only editors", async () => {
    const container = await renderEditor(true);

    fireEvent.dragOver(document.body, { dataTransfer: transfer(["Files"]) });

    await new Promise((r) => setTimeout(r, 20));
    expect(hint(container)).toBeNull();
  });

  it("clears the outline once the drag leaves the window", async () => {
    const container = await renderEditor();
    vi.useFakeTimers();

    fireEvent.dragOver(document.body, { dataTransfer: transfer(["Files"]) });
    expect(hint(container)).not.toBeNull();

    act(() => {
      vi.advanceTimersByTime(1000);
    });

    expect(hint(container)).toBeNull();
  });
});

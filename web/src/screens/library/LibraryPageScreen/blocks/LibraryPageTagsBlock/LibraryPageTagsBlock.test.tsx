import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DatagraphItemKind } from "@/api/openapi-schema";

import { LibraryPageTagsBlock } from "./LibraryPageTagsBlock";

let isDirectEditingMock = false;
let nodeIDMock = "node-123";
let tagsMock = [{ name: "prothetik", colour: "#e91e63" }];

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
  usePathname: () => "/l/test",
  useParams: () => ({ slug: ["test"] }),
}));

vi.mock("../../useEditState", () => ({
  useEditState: () => ({
    isDirectEditing: isDirectEditingMock,
  }),
}));

vi.mock("../../Context", () => ({
  useLibraryPageContext: () => ({
    nodeID: nodeIDMock,
    store: {
      getState: () => ({
        setTags: vi.fn(),
      }),
    },
  }),
}));

vi.mock("../../store", () => ({
  useWatch: (selector: (state: any) => unknown) =>
    selector({
      draft: {
        tags: tagsMock,
        content: "",
      },
    }),
}));

vi.mock("@/api/openapi-client/tags", () => ({
  tagGet: vi.fn(async (tagName: string) => ({
    id: "tag-1",
    name: tagName,
    colour: "#e91e63",
    items: [
      {
        kind: DatagraphItemKind.thread,
        ref: {
          id: "thread-101",
          title: "Prothetik Altklausur 2025",
          slug: "prothetik-altklausur-2025",
          body: "Fragen zur Prothetik Klausur",
          createdAt: "2025-01-01T00:00:00Z",
          author: {
            id: "user-1",
            name: "Max Mustermann",
            handle: "max",
            roles: [],
          },
        },
      },
    ],
  })),
}));

describe("LibraryPageTagsBlock", () => {
  beforeEach(() => {
    isDirectEditingMock = false;
    nodeIDMock = "node-123";
    tagsMock = [{ name: "prothetik", colour: "#e91e63" }];
  });

  it("renders tag badges and tagged posts in view mode", async () => {
    render(<LibraryPageTagsBlock />);

    expect(screen.getByText("prothetik")).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText("Prothetik Altklausur 2025")).toBeInTheDocument();
    });
  });

  it("renders tag picker and tagged posts in edit mode", async () => {
    isDirectEditingMock = true;

    render(<LibraryPageTagsBlock />);

    expect(screen.getByText("prothetik")).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText("Prothetik Altklausur 2025")).toBeInTheDocument();
    });
  });
});

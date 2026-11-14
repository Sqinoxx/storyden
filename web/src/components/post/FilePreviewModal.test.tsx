import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { recheckSession } from "@/api/client";
import { de } from "@/lib/i18n/translations/de";

import { FilePreviewModal } from "./FilePreviewModal";

vi.mock("@/api/client", () => ({ recheckSession: vi.fn() }));

const url = "http://localhost:8000/api/assets/z1-protokolle.pdf";

function stubFetch(response: Response | Error) {
  const fetchMock = vi.fn(async () => {
    if (response instanceof Error) throw response;
    return response;
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function renderPreview(displayName = "Z1 Protokolle.pdf") {
  return render(
    <FilePreviewModal
      url={url}
      displayName={displayName}
      mimeType={displayName.endsWith(".pdf") ? "application/pdf" : "text/plain"}
      onClose={() => {}}
      onDownload={() => {}}
    />,
  );
}

function problem(status: number) {
  return new Response(JSON.stringify({ title: "nope" }), {
    status,
    headers: { "Content-Type": "application/problem+json" },
  });
}

describe("FilePreviewModal", () => {
  beforeEach(() => {
    vi.mocked(recheckSession).mockReset();
    vi.stubGlobal("matchMedia", () => ({
      matches: false,
      addEventListener: () => {},
      removeEventListener: () => {},
    }));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows a sign-in prompt instead of the raw 401 body when the session has expired", async () => {
    const fetchMock = stubFetch(problem(401));

    renderPreview();

    expect(await screen.findByText(de.filePreview.sessionExpired)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: de.filePreview.signIn })).toHaveAttribute(
      "href",
      "/login",
    );
    expect(document.querySelector("iframe")).toBeNull();
    expect(recheckSession).toHaveBeenCalledTimes(1);

    expect(fetchMock).toHaveBeenCalledWith(
      url,
      expect.objectContaining({
        credentials: "include",
        headers: { Range: "bytes=0-0" },
      }),
    );
  });

  it("embeds the PDF once the asset is reachable", async () => {
    stubFetch(new Response("%", { status: 206 }));

    renderPreview();

    expect(await screen.findByTitle("Z1 Protokolle.pdf")).toHaveAttribute("src", url);
    expect(recheckSession).not.toHaveBeenCalled();
  });

  it("reports other failures without embedding the error body", async () => {
    stubFetch(problem(404));

    renderPreview();

    expect(await screen.findByText(de.filePreview.loadFailed)).toBeInTheDocument();
    expect(document.querySelector("iframe")).toBeNull();
  });

  it("still embeds links the probe cannot read, such as other sites blocked by CORS", async () => {
    stubFetch(new TypeError("Failed to fetch"));

    renderPreview();

    expect(await screen.findByTitle("Z1 Protokolle.pdf")).toBeInTheDocument();
  });

  it("does not render a 401 body as text file content", async () => {
    stubFetch(problem(401));

    renderPreview("notes.txt");

    expect(await screen.findByText(de.filePreview.sessionExpired)).toBeInTheDocument();
    expect(screen.queryByText(/nope/)).toBeNull();
  });
});

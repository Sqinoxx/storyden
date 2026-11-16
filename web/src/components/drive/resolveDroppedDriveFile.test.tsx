import { afterEach, describe, expect, it, vi } from "vitest";

import {
  fetchDriveDocument,
  filenameFromContentDisposition,
  importFromURL,
  isImportableFileURL,
  parseDriveDropURL,
  readDropURL,
} from "./resolveDroppedDriveFile";

vi.mock("@/config", () => ({
  API_ADDRESS: "https://api.unidentists.test",
  WEB_ADDRESS: "https://unidentists.test",
}));

const assetImport = vi.fn();
vi.mock("@/api/openapi-client/assets", () => ({
  assetImport: (body: unknown) => assetImport(body),
}));

const driveFolderList = vi.fn();
vi.mock("@/api/openapi-client/drive", () => ({
  driveFolderList: () => driveFolderList(),
}));

const GOOGLE_ID = "1wLKOp83T2tUPw5neR2qhioik8UQLX5so";
const FOLDER = "dbf0dtosi88000bln04g";
const PDF = "application/pdf";
const DOCX =
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document";
const PPTX =
  "application/vnd.openxmlformats-officedocument.presentationml.presentation";

function response(body: string, headers: Record<string, string>, status = 200) {
  return new Response(body, { status, headers });
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("parseDriveDropURL", () => {
  it.each([
    [
      "site drive page",
      `https://unidentists.test/drive/${FOLDER}/${GOOGLE_ID}`,
      { folderID: FOLDER, fileID: GOOGLE_ID },
    ],
    [
      "proxied content url",
      `https://api.unidentists.test/api/drive/folders/${FOLDER}/files/${GOOGLE_ID}/content?disposition=inline`,
      { folderID: FOLDER, fileID: GOOGLE_ID },
    ],
    [
      "google drive file link",
      `https://drive.google.com/file/d/${GOOGLE_ID}/view?usp=drivesdk`,
      { fileID: GOOGLE_ID },
    ],
    [
      "google drive open link",
      `https://drive.google.com/open?id=${GOOGLE_ID}`,
      { fileID: GOOGLE_ID },
    ],
    [
      "google docs link",
      `https://docs.google.com/document/d/${GOOGLE_ID}/edit`,
      { fileID: GOOGLE_ID },
    ],
  ])("recognises a %s", (_, url, expected) => {
    expect(parseDriveDropURL(url)).toEqual(expected);
  });

  it.each([
    ["a foreign site", `https://example.com/drive/${FOLDER}/${GOOGLE_ID}`],
    ["a registered folder root", `https://unidentists.test/drive/${FOLDER}`],
    ["a thread", "https://unidentists.test/t/some-thread"],
    ["a google folder", "https://drive.google.com/drive/my-drive"],
    ["plain text", "not a url"],
    ["nothing", undefined],
  ])("ignores %s", (_, url) => {
    expect(parseDriveDropURL(url)).toBeNull();
  });
});

describe("readDropURL", () => {
  function transfer(data: Record<string, string>) {
    return {
      types: Object.keys(data),
      getData: (type: string) => data[type] ?? "",
    } as unknown as DataTransfer;
  }

  it("reads a google drive desktop drag from DownloadURL", () => {
    expect(
      readDropURL(
        transfer({
          DownloadURL: `application/pdf:2010 WS.pdf:https://drive.google.com/uc?id=${GOOGLE_ID}&export=download`,
          "text/plain": "2010 WS.pdf",
        }),
      ),
    ).toBe(`https://drive.google.com/uc?id=${GOOGLE_ID}&export=download`);
  });

  it("prefers a file link over the page link", () => {
    expect(
      readDropURL(
        transfer({
          "text/uri-list": "https://drive.google.com/drive/my-drive",
          "text/html": `<a href="https://drive.google.com/file/d/${GOOGLE_ID}/view?usp=drive_web&amp;x=1">2010 WS.pdf</a>`,
        }),
      ),
    ).toBe(`https://drive.google.com/file/d/${GOOGLE_ID}/view?usp=drive_web&x=1`);
  });

  it("finds an image in dragged html", () => {
    expect(
      readDropURL(
        transfer({
          "text/uri-list": "https://example.com/gallery",
          "text/html": `<img src="https://cdn.example.com/Zahn.JPG" alt="">`,
        }),
      ),
    ).toBe("https://cdn.example.com/Zahn.JPG");
  });

  it("scans unknown drag types for a file link", () => {
    expect(
      readDropURL(
        transfer({
          "application/x-drive-item": `{"url":"https://drive.google.com/open?id=${GOOGLE_ID}"}`,
        }),
      ),
    ).toBe(`https://drive.google.com/open?id=${GOOGLE_ID}`);
  });

  it("prefers the first uri-list entry over comments", () => {
    expect(
      readDropURL(
        transfer({
          "text/uri-list": "# comment\r\nhttps://a.test/x\r\nhttps://b.test",
          "text/plain": "https://c.test",
        }),
      ),
    ).toBe("https://a.test/x");
  });

  it("falls back to a single plain text url", () => {
    expect(readDropURL(transfer({ "text/plain": " https://c.test " }))).toBe(
      "https://c.test",
    );
  });

  it("ignores prose", () => {
    expect(
      readDropURL(transfer({ "text/plain": "see https://c.test" })),
    ).toBeUndefined();
  });
});

describe("filenameFromContentDisposition", () => {
  it("prefers the encoded filename", () => {
    expect(
      filenameFromContentDisposition(
        `attachment; filename="Aspirin.pdf"; filename*=UTF-8''Aspirin%20%C3%BCbersicht.pdf`,
      ),
    ).toBe("Aspirin übersicht.pdf");
  });

  it("reads a quoted filename", () => {
    expect(
      filenameFromContentDisposition(`inline; filename="Diclofenac.pdf"`),
    ).toBe("Diclofenac.pdf");
  });

  it("returns undefined without a header", () => {
    expect(filenameFromContentDisposition(null)).toBeUndefined();
  });
});

describe("fetchDriveDocument", () => {
  it("returns a pdf as a named file", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      response("%PDF-1.4", {
        "content-type": PDF,
        "content-disposition": `attachment; filename="Aspirin.pdf"`,
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const file = await fetchDriveDocument({
      folderID: FOLDER,
      fileID: GOOGLE_ID,
    });

    expect(file?.name).toBe("Aspirin.pdf");
    expect(file?.type).toBe(PDF);
    expect(fetchMock).toHaveBeenCalledWith(
      `https://api.unidentists.test/api/drive/folders/${FOLDER}/files/${GOOGLE_ID}/content?disposition=attachment`,
      { credentials: "include" },
    );
  });

  it("accepts word documents", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          response("docx", { "content-type": `${DOCX}; charset=binary` }),
        ),
    );

    const file = await fetchDriveDocument({
      folderID: FOLDER,
      fileID: GOOGLE_ID,
    });

    expect(file?.type).toBe(DOCX);
    expect(file?.name).toBe("dokument.docx");
  });

  it("rejects other file types", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(response("hi", { "content-type": "text/plain" })),
    );

    expect(
      await fetchDriveDocument({ folderID: FOLDER, fileID: GOOGLE_ID }),
    ).toBeNull();
  });

  it("rejects folders and missing files", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          response("{}", { "content-type": "application/json" }, 400),
        ),
    );

    expect(
      await fetchDriveDocument({ folderID: FOLDER, fileID: GOOGLE_ID }),
    ).toBeNull();
  });

  it("finds a google link in whichever registered folder holds it", async () => {
    driveFolderList.mockResolvedValue({
      configured: true,
      folders: [{ id: "folder-a" }, { id: "folder-b" }],
    });

    const fetchMock = vi.fn((url: string) =>
      Promise.resolve(
        url.includes("folder-b")
          ? response("%PDF", { "content-type": PDF })
          : response("{}", { "content-type": "application/json" }, 404),
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    const file = await fetchDriveDocument({ fileID: GOOGLE_ID });

    expect(file?.type).toBe(PDF);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});

describe("isImportableFileURL", () => {
  it.each([
    "https://register.awmf.org/assets/guidelines/083-001l_S3_Kariesprophylaxe.pdf",
    "https://example.com/files/Vorlesung%20Endo.PPTX",
    "https://example.com/Lernliste.xlsx?download=1",
    "https://example.com/protokoll.docx",
    "https://example.com/photo.jpg",
    "https://example.com/Befund.PNG",
    "https://example.com/scan.webp",
    "https://example.com/IMG_0001.heic",
    `https://drive.google.com/file/d/${GOOGLE_ID}/view`,
    `https://docs.google.com/presentation/d/${GOOGLE_ID}/edit`,
  ])("accepts %s", (url) => {
    expect(isImportableFileURL(url)).toBe(true);
  });

  it.each([
    ["an article", "https://www.zm-online.de/news/aktuelles"],
    ["an svg", "https://example.com/logo.svg"],
    [
      "the site's own drive",
      `https://unidentists.test/drive/${FOLDER}/${GOOGLE_ID}`,
    ],
    ["the site's own files", "https://unidentists.test/files/skript.pdf"],
    ["a google folder", "https://drive.google.com/drive/my-drive"],
    ["a non-http link", "ftp://example.com/a.pdf"],
    ["nothing", undefined],
  ])("rejects %s", (_, url) => {
    expect(isImportableFileURL(url)).toBe(false);
  });
});

describe("importFromURL", () => {
  it("returns the imported asset", async () => {
    const asset = { id: "a", filename: "Leitlinie.pdf", path: "/api/assets/a" };
    assetImport.mockResolvedValue(asset);

    expect(await importFromURL("https://example.com/Leitlinie.pdf")).toBe(
      asset,
    );
    expect(assetImport).toHaveBeenCalledWith({
      url: "https://example.com/Leitlinie.pdf",
    });
  });

  it("returns null when the server refuses", async () => {
    assetImport.mockRejectedValue(new Error("400"));

    expect(await importFromURL("https://example.com/Leitlinie.pdf")).toBeNull();
  });
});

describe("fetchDriveDocument images", () => {
  it("accepts photos from the drive", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(response("jpg", { "content-type": "image/jpeg" })),
    );

    const file = await fetchDriveDocument({
      folderID: FOLDER,
      fileID: GOOGLE_ID,
    });

    expect(file?.type).toBe("image/jpeg");
    expect(file?.name).toBe("dokument.jpg");
  });
});

describe("fetchDriveDocument slides", () => {
  it("accepts powerpoint files from the drive", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(response("pptx", { "content-type": PPTX })),
    );

    const file = await fetchDriveDocument({
      folderID: FOLDER,
      fileID: GOOGLE_ID,
    });

    expect(file?.type).toBe(PPTX);
    expect(file?.name).toBe("dokument.pptx");
  });
});

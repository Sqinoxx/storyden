import { describe, expect, it } from "vitest";

import { fileExtension, getFileTypeInfo } from "./fileType";

describe("getFileTypeInfo", () => {
  it.each([
    ["application/pdf", "Befund.pdf", "pdf", "PDF"],
    [undefined, "Brief.docx", "word", "DOCX"],
    [undefined, "Abrechnung.XLSX", "excel", "XLSX"],
    ["text/csv", "export.csv", "csv", "CSV"],
    [undefined, "Vortrag.pptx", "powerpoint", "PPTX"],
    ["application/zip", "scans.zip", "archive", "ZIP"],
    ["image/jpeg", "opg.jpg", "image", "JPG"],
    ["video/mp4", "clip.mp4", "video", "MP4"],
    ["audio/mpeg", "memo.mp3", "audio", "MP3"],
    ["text/plain", "notes.txt", "text", "TXT"],
  ])("%s %s → %s", (mime, name, kind, label) => {
    expect(getFileTypeInfo(mime, name)).toEqual({ kind, label });
  });

  it("falls back to the mime type when there is no extension", () => {
    expect(getFileTypeInfo("application/pdf", "Befund")).toEqual({
      kind: "pdf",
      label: "PDF",
    });
    expect(
      getFileTypeInfo(
        "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
        undefined,
      ),
    ).toEqual({ kind: "word", label: "DOC" });
  });

  it("uses the kind label for long extensions", () => {
    expect(getFileTypeInfo(undefined, "Kalkulation.numbers")).toEqual({
      kind: "excel",
      label: "XLS",
    });
  });

  it("returns a generic file without label for unknown types", () => {
    expect(getFileTypeInfo(undefined, "README")).toEqual({
      kind: "file",
      label: "",
    });
    expect(getFileTypeInfo("application/octet-stream", "data.bin")).toEqual({
      kind: "file",
      label: "BIN",
    });
  });
});

describe("fileExtension", () => {
  it("ignores dotfiles, trailing dots and query strings", () => {
    expect(fileExtension(".env")).toBe("");
    expect(fileExtension("file.")).toBe("");
    expect(fileExtension("a.pdf?x=1")).toBe("pdf");
  });
});

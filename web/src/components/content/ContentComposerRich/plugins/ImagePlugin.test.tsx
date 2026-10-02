import { Editor } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import { describe, expect, it } from "vitest";

import { FileAttachmentExtended } from "./FileAttachmentPlugin";
import { ImageExtended } from "./ImagePlugin";

function editor(content: string) {
  return new Editor({
    extensions: [StarterKit, ImageExtended, FileAttachmentExtended],
    content,
  });
}

describe("ImageExtended", () => {
  it("keeps a file inserted after an image outside of the image", () => {
    const e = editor("<p></p>");
    const { schema } = e.state;

    const image = schema.nodes["image"]!.create({ src: "/a.png" });
    const file = schema.nodes["fileAttachment"]!.create({
      href: "/api/assets/b.pdf",
      fileName: "b.pdf",
    });

    let pos = e.state.selection.$head.pos;
    e.view.dispatch(e.state.tr.insert(pos, image));
    pos += image.nodeSize;
    e.view.dispatch(e.state.tr.insert(pos, file));

    const imageNode = e.state.doc.content.content.find(
      (n) => n.type.name === "image",
    );
    expect(imageNode?.childCount ?? 0).toBe(0);
    expect(e.getHTML()).toContain('data-filename="b.pdf"');
  });

  it("parses an attachment after an image as a sibling", () => {
    const e = editor(
      '<p><img src="/a.png"></p><p><a href="/api/assets/b.pdf" data-type="file-attachment" data-filename="b.pdf">b.pdf</a></p>',
    );

    let inside = false;
    e.state.doc.descendants((node, _pos, parent) => {
      if (node.type.name === "fileAttachment" && parent?.type.name === "image") {
        inside = true;
      }
    });
    expect(inside).toBe(false);
  });
});

import { describe, expect, it } from "vitest";

import { slugify } from "@/utils/slugify";

import { FormSchema } from "./useProfileScreen";

const base = { name: "Stefan", bio: "", signature: "" };

describe("profile FormSchema handle", () => {
  it("accepts a lowercase handle", () => {
    expect(
      FormSchema.safeParse({ ...base, handle: "stefan_p-1" }).success,
    ).toBe(true);
  });

  it.each(["Stefan", "stefan p", "stefan!"])(
    "rejects %s with a message explaining the allowed characters",
    (handle) => {
      const result = FormSchema.safeParse({ ...base, handle });
      expect(result.success).toBe(false);
      expect(result.error?.issues[0]?.message).toMatch(/lowercase/);
      expect(
        FormSchema.safeParse({ ...base, handle: slugify(handle) }).success,
      ).toBe(true);
    },
  );
});

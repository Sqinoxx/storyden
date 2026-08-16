import z from "zod";

import { Account } from "@/api/openapi-schema";

import { EditorSettingsSchema } from "./editor";
import { FrontendConfiguration, Settings } from "./settings";
import { SidebarSettingsSchema } from "./sidebar";

export const SessionSettingsSchema = z.object({
  // How long a "remember me" session lasts, in days. Clamped server-side to
  // the instance maximum.
  remember_me_days: z.number().int().min(1).max(365),
});
export type SessionSettings = z.infer<typeof SessionSettingsSchema>;

const MemberCustomSettingsParseSchema = z.object({
  editor: EditorSettingsSchema.optional(),
  sidebar: SidebarSettingsSchema.optional(),
  session: SessionSettingsSchema.optional(),
});

export const MemberCustomSettingsSchema = z.object({
  editor: EditorSettingsSchema,
  sidebar: SidebarSettingsSchema,
  session: SessionSettingsSchema,
});
export type MemberCustomSettings = z.infer<typeof MemberCustomSettingsSchema>;

// Member extends Account with custom member settings by typing `meta`.
export type Member = Account & {
  meta: MemberCustomSettings;
};

export const DefaultMemberSettings: MemberCustomSettings = {
  editor: {
    mode: "richtext",
  },
  sidebar: {
    defaultState: "closed",
  },
  session: {
    remember_me_days: 30,
  },
};

export function parseMemberSettings(
  data: Account,
  global?: FrontendConfiguration,
): Member {
  const parsed = MemberCustomSettingsParseSchema.safeParse(data.meta ?? {});

  const rawMeta = parsed.success ? parsed.data : {};
  if (!parsed.success && data.meta) {
    console.warn("Failed to parse member settings meta:", parsed.error);
  }

  const meta: MemberCustomSettings = {
    editor: {
      mode:
        rawMeta.editor?.mode ??
        global?.editor.mode ??
        DefaultMemberSettings.editor.mode,
    },
    sidebar: {
      defaultState:
        rawMeta.sidebar?.defaultState ??
        global?.sidebar.defaultState ??
        DefaultMemberSettings.sidebar.defaultState,
    },
    session: {
      remember_me_days:
        rawMeta.session?.remember_me_days ??
        DefaultMemberSettings.session.remember_me_days,
    },
  };

  const settings = { ...data, meta } satisfies Member;

  return settings;
}

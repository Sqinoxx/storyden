import { z } from "zod";

import { getAssetURL } from "@/utils/asset";

export const FALLBACK_LOGO_URL = "/StuDent-transparent.png";

export const LogoSettingsSchema = z.object({
  light: z.string().default(""),
  dark: z.string().default(""),
});
export type LogoSettings = z.infer<typeof LogoSettingsSchema>;

export const DefaultLogoSettings: LogoSettings = {
  light: "",
  dark: "",
};

export type LogoVariant = keyof LogoSettings;

export function getLogoURLs(logo?: LogoSettings): LogoSettings {
  const light = logo?.light || logo?.dark;
  const dark = logo?.dark || logo?.light;

  return {
    light: light ? getAssetURL(light) : FALLBACK_LOGO_URL,
    dark: dark ? getAssetURL(dark) : FALLBACK_LOGO_URL,
  };
}

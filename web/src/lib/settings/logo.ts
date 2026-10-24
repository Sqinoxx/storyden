import { z } from "zod";

import { getAssetURL } from "@/utils/asset";

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
    light: getAssetURL(light),
    dark: getAssetURL(dark),
  };
}

import { RegistrationMode } from "@/api/openapi-schema";

import type { Settings } from "./settings";
import { useSettings } from "./settings-client";

export function allowsPublicRegistration(mode?: RegistrationMode) {
  return mode === undefined || mode === RegistrationMode.public;
}

export function usePublicRegistration(fallbackData?: Settings) {
  const result = useSettings(fallbackData);

  if (!result.ready) {
    return fallbackData
      ? allowsPublicRegistration(fallbackData.registration_mode)
      : false;
  }

  return allowsPublicRegistration(result.settings.registration_mode);
}

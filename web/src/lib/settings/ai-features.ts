import { AIFeatureFlags } from "@/api/openapi-schema";

import { useSettings } from "./settings-client";

export type AIFeature = keyof AIFeatureFlags;

export function useAIFeature(feature: AIFeature) {
  const { settings } = useSettings();

  return settings?.ai_features?.[feature] ?? false;
}

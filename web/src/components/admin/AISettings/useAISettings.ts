import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import {
  adminGenerateTagMeanings,
  useAdminOCRStats,
} from "@/api/openapi-client/admin";
import { useTranslation } from "@/lib/i18n";
import { useCapability } from "@/lib/settings/capabilities";
import { useSettingsMutation } from "@/lib/settings/mutation";
import { AdminSettings } from "@/lib/settings/settings";

export type Props = {
  settings: AdminSettings;
};

export const FormSchema = z.object({
  threadSummary: z.boolean(),
  tagSuggestion: z.boolean(),
  semesterSuggestion: z.boolean(),
  libraryAssist: z.boolean(),
  pdfSummary: z.boolean(),
  pdfSummaryBackground: z.boolean(),
  pdfSummaryNightOnly: z.boolean(),
  tagMeanings: z.string().max(10000),
});
export type Form = z.infer<typeof FormSchema>;

export function useAISettings({ settings }: Props) {
  const t = useTranslation();
  const configured = useCapability("gen_ai");
  const { revalidate, updateSettings } = useSettingsMutation();
  const [generating, setGenerating] = useState(false);
  const [draftResult, setDraftResult] = useState<string | null>(null);

  const ai = settings.services?.ai;
  const { data: stats } = useAdminOCRStats();
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      threadSummary: ai?.thread_summary ?? false,
      tagSuggestion: ai?.tag_suggestion ?? true,
      semesterSuggestion: ai?.semester_suggestion ?? true,
      libraryAssist: ai?.library_assist ?? true,
      pdfSummary: ai?.pdf_summary ?? false,
      pdfSummaryBackground: ai?.pdf_summary_background ?? true,
      pdfSummaryNightOnly: ai?.pdf_summary_night_only ?? false,
      tagMeanings: settings.metadata.ai.tagMeanings,
    },
  });

  const onSubmit = form.handleSubmit(async (data) => {
    await handle(
      async () => {
        await updateSettings({
          metadata: {
            ...settings.metadata,
            ai: {
              tagMeanings: data.tagMeanings.trim(),
            },
          },
          services: {
            ai: {
              thread_summary: data.threadSummary,
              tag_suggestion: data.tagSuggestion,
              semester_suggestion: data.semesterSuggestion,
              library_assist: data.libraryAssist,
              pdf_summary: data.pdfSummary,
              pdf_summary_background: data.pdfSummaryBackground,
              pdf_summary_night_only: data.pdfSummaryNightOnly,
            },
          },
        });
      },
      {
        promiseToast: {
          loading: t.toasts.savingSettings,
          success: t.toasts.settingsSaved,
        },
        cleanup: async () => {
          await revalidate();
        },
      },
    );
  });

  const generateMeanings = async () => {
    setGenerating(true);
    setDraftResult(null);
    await handle(
      async () => {
        const draft = await adminGenerateTagMeanings();
        form.setValue("tagMeanings", draft.meanings, { shouldDirty: true });
        setDraftResult(
          draft.added.length === 0 && draft.skipped.length === 0
            ? t.aiSettings.tagMeaningsNothing
            : t.aiSettings.tagMeaningsResult
                .replace("{added}", String(draft.added.length))
                .replace("{skipped}", String(draft.skipped.length)),
        );
      },
      {
        cleanup: async () => {
          setGenerating(false);
        },
      },
    );
  };

  const pdfProgress =
    stats?.pdf_summaries_total !== undefined
      ? t.aiSettings.pdfSummaryProgress
          .replace("{done}", String(stats.pdf_summaries_done ?? 0))
          .replace("{total}", String(stats.pdf_summaries_total))
      : null;

  return {
    configured,
    pdfProgress,
    watch: form.watch,
    control: form.control,
    register: form.register,
    formState: form.formState,
    generating,
    draftResult,
    generateMeanings,
    onSubmit,
  };
}

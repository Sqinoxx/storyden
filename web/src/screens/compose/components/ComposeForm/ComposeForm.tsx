"use client";

import { FormProvider } from "react-hook-form";

import { CategorySelect } from "@/components/category/CategorySelect/CategorySelect";
import { TagListField } from "@/components/thread/ThreadTagList";
import { Button } from "@/components/ui/button";
import { HStack, WStack, styled } from "@/styled-system/jsx";
import { useTranslation } from "@/lib/i18n";

import { BodyInput } from "../BodyInput/BodyInput";
import { TitleInput } from "../TitleInput/TitleInput";

import { Props, useComposeForm } from "./useComposeForm";

import { AssetUploadAction } from "@/components/asset/AssetUploadAction";

export function ComposeForm(props: Props) {
  const { form, state, handlers } = useComposeForm(props);
  const t = useTranslation();

  return (
    <styled.form
      display="flex"
      flexDir="column"
      alignItems="start"
      w="full"
      h="full"
      gap="2"
      onSubmit={handlers.handlePublish}
    >
      <FormProvider {...form}>
        <WStack
          flexDir={{
            base: "column-reverse",
            md: "row",
          }}
          alignItems={{
            base: "end",
            md: "center",
          }}
        >
          <HStack width="full" flexWrap="wrap">
            <CategorySelect control={form.control} name="category" />
            <TagListField
              name="tags"
              control={form.control}
              initialTags={props.initialDraft?.tags}
            />
            <AssetUploadAction
              title="Datei anhängen"
              operation="add"
              accept={[
                "application/pdf",
                "application/msword",
                "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
              ]}
              onFinish={handlers.handleAttach}
            >
              <Button type="button" variant="outline" size="xs">
                Datei anhängen (PDF/Word)
              </Button>
            </AssetUploadAction>
          </HStack>

          <HStack flexShrink={0}>
            <Button
              variant="ghost"
              size="xs"
              disabled={!form.formState.isValid || state.isSavingDraft}
              onClick={handlers.handleSaveDraft}
              loading={state.isSavingDraft}
            >
              {t.editor.saveDraft}
            </Button>

            <Button
              variant="subtle"
              size="xs"
              type="submit"
              disabled={!form.formState.isValid || state.isPublishing}
              loading={state.isPublishing}
            >
              {t.editor.post}
            </Button>
          </HStack>
        </WStack>

        {state.attachments.length > 0 && (
          <HStack width="full" flexWrap="wrap" gap="2" pt="1">
            {state.attachments.map((a) => (
              <HStack
                key={a.id}
                gap="1.5"
                px="2"
                py="1"
                borderRadius="md"
                bg="bg.muted"
                borderWidth="thin"
                borderStyle="solid"
                borderColor="border.subtle"
                fontSize="xs"
              >
                <span>{a.filename}</span>
                <Button
                  type="button"
                  variant="ghost"
                  size="xs"
                  onClick={() => handlers.handleDetach(a)}
                >
                  entfernen
                </Button>
              </HStack>
            ))}
          </HStack>
        )}

        <HStack width="full" justifyContent="space-between" alignItems="start">
          <HStack width="full">
            <TitleInput />
          </HStack>
        </HStack>

        <BodyInput onAssetUpload={() => handlers.handleAssetUpload()} />
      </FormProvider>
    </styled.form>
  );
}

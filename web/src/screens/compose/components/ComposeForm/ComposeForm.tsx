import { FormProvider } from "react-hook-form";

import { CategorySelect } from "@/components/category/CategorySelect/CategorySelect";
import { TagListField } from "@/components/thread/ThreadTagList";
import { Button } from "@/components/ui/button";
import { HStack, WStack, styled } from "@/styled-system/jsx";

import { BodyInput } from "../BodyInput/BodyInput";
import { TitleInput } from "../TitleInput/TitleInput";

import { Props, useComposeForm } from "./useComposeForm";

import { AssetUploadAction } from "@/components/asset/AssetUploadAction";

export function ComposeForm(props: Props) {
  const { form, state, handlers } = useComposeForm(props);

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
          <HStack width="full">
            <CategorySelect control={form.control} name="category" />
            <TagListField
              name="tags"
              control={form.control}
              initialTags={props.initialDraft?.tags}
            />
          </HStack>

          <HStack>
            <Button
              variant="ghost"
              size="xs"
              disabled={!form.formState.isValid || state.isSavingDraft}
              onClick={handlers.handleSaveDraft}
              loading={state.isSavingDraft}
            >
              Save draft
            </Button>

            <Button
              variant="subtle"
              size="xs"
              type="submit"
              disabled={!form.formState.isValid || state.isPublishing}
              loading={state.isPublishing}
            >
              Post
            </Button>
          </HStack>
        </WStack>

        <HStack width="full" justifyContent="space-between" alignItems="start">
          <HStack width="full">
            <TitleInput />
          </HStack>
        </HStack>

        <BodyInput onAssetUpload={() => handlers.handleAssetUpload()} />
        <HStack width="full" flexWrap="wrap">
  {state.attachments.map((a) => (
    <HStack key={a.id} gap="1">
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

  <AssetUploadAction
  title="Datei anhängen"
  operation="add"
  accept={["application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"]}
  onFinish={handlers.handleAttach}
>
  <Button type="button" variant="outline" size="xs">
    Datei anhängen (PDF/Word)
  </Button>
</AssetUploadAction>
</HStack>
      </FormProvider>
    </styled.form>
  );
}

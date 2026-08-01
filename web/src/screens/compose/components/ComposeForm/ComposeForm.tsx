"use client";

import { FormProvider } from "react-hook-form";

import { CategorySelect } from "@/components/category/CategorySelect/CategorySelect";
import { TagListField } from "@/components/thread/ThreadTagList";
import { Button } from "@/components/ui/button";
import { Box, CardBox, HStack, WStack, styled } from "@/styled-system/jsx";
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
      gap="4"
      onSubmit={handlers.handlePublish}
    >
      <FormProvider {...form}>
        {/* Outer Master Card Box surrounding both inner boxes */}
        <CardBox
          p={{ base: "3", md: "4" }}
          w="full"
          borderStyle="solid"
          borderWidth="thin"
          borderColor="border.subtle"
          borderRadius="2xl"
          bg="bg.subtle"
          gap="4"
        >
          {/* Card Box 1: Toolbar / Options & Actions */}
          <CardBox
            p="4"
            w="full"
            borderStyle="solid"
            borderWidth="thin"
            borderColor="border.subtle"
            borderRadius="xl"
            bg="bg.default"
          >
            <WStack
              flexDir={{
                base: "column-reverse",
                md: "row",
              }}
              alignItems={{
                base: "end",
                md: "center",
              }}
              gap="3"
            >
              <HStack width="full" flexWrap="wrap" gap="2">
                <CategorySelect control={form.control} name="category" />
                <TagListField
                  name="tags"
                  control={form.control}
                  initialTags={props.initialDraft?.tags}
                />
              </HStack>

              <HStack flexShrink={0} gap="2">
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
          </CardBox>

          {/* Card Box 2: Post Composition / Title & Content Editor */}
          <CardBox
            p={{ base: "4", md: "6" }}
            w="full"
            borderStyle="solid"
            borderWidth="thin"
            borderColor="border.subtle"
            borderRadius="xl"
            bg="bg.default"
            minH="[450px]"
            display="flex"
            flexDir="column"
            gap="4"
          >
            <Box w="full" pb="3" borderBottomWidth="thin" borderColor="border.subtle">
              <TitleInput />
            </Box>

            <Box w="full" flex="1">
              <BodyInput onAssetUpload={() => handlers.handleAssetUpload()} />
            </Box>

            {/* Footer row inside Box 2: Attachments & Bottom-Right "Datei anhängen" button */}
            <WStack
              w="full"
              pt="3"
              borderTopWidth="thin"
              borderColor="border.subtle"
              alignItems="center"
              justifyContent="space-between"
              flexWrap="wrap"
              gap="2"
            >
              <HStack flexWrap="wrap" gap="2">
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

              <Box ml="auto">
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
              </Box>
            </WStack>
          </CardBox>
        </CardBox>
      </FormProvider>
    </styled.form>
  );
}

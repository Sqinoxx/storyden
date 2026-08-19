"use client";

import { FormProvider } from "react-hook-form";

import { CategoryTreeSelect } from "@/components/category/CategoryTreeSelect/CategoryTreeSelect";
import { ContentDragOverlay } from "@/components/content/ContentDragOverlay";
import { SemesterSelect } from "@/components/thread/SemesterSelect/SemesterSelect";
import { TagListField } from "@/components/thread/ThreadTagList";
import { Button } from "@/components/ui/button";
import { Box, CardBox, HStack, WStack, styled } from "@/styled-system/jsx";
import { useTranslation } from "@/lib/i18n";
import { getCleanFilename } from "@/utils/asset";

import { BodyInput } from "../BodyInput/BodyInput";
import { TitleInput } from "../TitleInput/TitleInput";

import { Props, useComposeForm } from "./useComposeForm";
import { useCardDragDrop } from "./useCardDragDrop";

import { AssetUploadAction } from "@/components/asset/AssetUploadAction";

export function ComposeForm(props: Props) {
  const { form, state, handlers } = useComposeForm(props);
  const cardDragDrop = useCardDragDrop(handlers.handleAttach);
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
              flexDirection={{
                base: "column",
                md: "row",
              }}
              alignItems={{
                base: "stretch",
                md: "center",
              }}
              gap="3"
            >
              <HStack width="full" flexWrap="wrap" gap="2">
                <CategoryTreeSelect control={form.control} name="category" />
                <HStack
                  flex="1"
                  minW="[220px]"
                  gap="2"
                  flexDirection={{ base: "column", md: "row" }}
                  alignItems={{ base: "stretch", md: "center" }}
                >
                  <Box flex="1" minW="0">
                    <SemesterSelect control={form.control} name="semester" />
                  </Box>
                  <Box flex="1" minW="0">
                    <TagListField
                      name="tags"
                      control={form.control}
                      initialTags={props.initialDraft?.tags}
                      attachments={state.attachments}
                      triggerProps={{ w: "full", minW: "0", maxW: "full" }}
                    />
                  </Box>
                </HStack>
              </HStack>

              <HStack flexShrink={0} gap="2" justifyContent={{ base: "flex-end", md: "flex-start" }} width={{ base: "full", md: "auto" }}>
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
            position="relative"
            onDragOver={cardDragDrop.handlers.handleDragOver}
            onDragEnter={cardDragDrop.handlers.handleDragEnter}
            onDragLeave={cardDragDrop.handlers.handleDragLeave}
            onDrop={cardDragDrop.handlers.handleDrop}
          >
            <Box w="full" pb="3" borderBottomWidth="thin" borderColor="border.subtle">
              <TitleInput />
            </Box>

            <Box w="full" flex="1">
              <BodyInput onAssetUpload={handlers.handleAttach} />
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
                    <span>{getCleanFilename(a.filename)}</span>
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
                  onFinish={handlers.handleAttach}
                >
                  <Button type="button" variant="outline" size="xs">
                    Datei anhängen
                  </Button>
                </AssetUploadAction>
              </Box>
            </WStack>

            {cardDragDrop.isDragging && (
              <ContentDragOverlay
                isError={cardDragDrop.isDragError}
                message={cardDragDrop.getDragOverlayMessage()}
              />
            )}
          </CardBox>
        </CardBox>
      </FormProvider>
    </styled.form>
  );
}

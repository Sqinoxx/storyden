import { useId } from "react";

import { match } from "ts-pattern";
import { X } from "lucide-react";

import { LinkReference } from "@/api/openapi-schema";
import { useAttachmentUpload } from "@/components/content/useAttachmentUpload";
import { CategorySelect } from "@/components/category/CategorySelect/CategorySelect";
import { TagListField } from "@/components/thread/ThreadTagList";
import { FileAttachmentBadge } from "@/components/post/FileAttachmentList";
import { Spinner } from "@/components/ui/Spinner";
import { Button } from "@/components/ui/button";
import { ComposeField } from "@/components/ui/form/ComposeField";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { Input } from "@/components/ui/input";
import { CreateIcon } from "@/components/ui/icons/Create";
import { Card } from "@/components/ui/rich-card";
import { Box, CardBox, HStack, WStack, styled } from "@/styled-system/jsx";
import { button } from "@/styled-system/recipes";
import { lstack } from "@/styled-system/patterns";
import { useTranslation } from "@/lib/i18n";
import { getAssetURL } from "@/utils/asset";

import { Props, useQuickShare } from "./useQuickShare";

export function QuickShare(props: Props) {
  const fileInputId = useId();
  const { isUploading, upload } = useAttachmentUpload();

  const {
    form,
    state: { formRef, editing, hydratedLink, resetKey, uploadedAssets },
    handlers,
  } = useQuickShare(props);
  const t = useTranslation();

  // TODO: Render a prompt to sign up to contribute if not logged in.
  if (!props.initialSession) {
    return null;
  }

  return (
    <CardBox
      bgColor="bg.default"
      padding={editing ? "3" : "2"}
      boxShadow={editing ? "md" : "sm"}
      transitionDuration="slower"
      transitionTimingFunction="default"
      style={{ transitionProperty: "padding, box-shadow" }}
    >
      <styled.form
        className={lstack({
          gap: "2",
        })}
        ref={formRef}
        onFocus={handlers.handleFocus}
        onSubmit={handlers.handlePost}
        minHeight={editing ? "48" : "0"}
        transitionDuration="slower"
        transitionTimingFunction="default"
        style={{ transitionProperty: "min-height" }}
      >
        <Input
          {...form.register("title")}
          placeholder={t.editor.titlePlaceholder}
          fontWeight="semibold"
          fontSize="md"
          px="3"
          py="2"
          h="auto"
          w="full"
          borderRadius="md"
          borderColor="border.subtle"
          bg="bg.subtle/30"
          _focus={{
            borderColor: "accent.default",
            bg: "bg.default",
          }}
        />

        <Box flex="1" minH="0" w="full">
          <ComposeField
            control={form.control}
            name="body"
            placeholder={t.feed.quickSharePlaceholder}
            resetKey={resetKey}
          />
        </Box>

        <WStack
          w="full"
          justifyContent="space-between"
          alignItems="center"
          gap="2"
          flexWrap="wrap"
        >
          <HStack alignItems="center" flexWrap="wrap" gap="2" flex="1" minW="0">
            {props.showCategorySelect && (
              <>
                <CategorySelect control={form.control} name="category" />

                <FormErrorText>
                  {form.formState.errors["category"]?.message}
                </FormErrorText>
              </>
            )}

            <TagListField
              name="tags"
              control={form.control}
              attachments={uploadedAssets}
            />
          </HStack>

          <HStack gap="2" ml="auto" flexShrink={0}>
            <styled.label
              className={button({ size: "sm", variant: "ghost" })}
              htmlFor={fileInputId}
              title={t.editor.uploadFile}
              aria-disabled={isUploading}
              opacity={isUploading ? "5" : "full"}
              pointerEvents={isUploading ? "none" : "auto"}
            >
              📎 {t.editor.uploadFile}
            </styled.label>
            <styled.input
              id={fileInputId}
              type="file"
              multiple
              display="none"
              accept="*"
              disabled={isUploading}
              onChange={async (e) => {
                const files = Array.from(e.currentTarget.files ?? []);
                e.currentTarget.value = "";

                const uploaded = await upload(files);

                for (const { markup } of uploaded) {
                  const current = form.getValues("body") ?? "";
                  form.setValue("body", current + "<p>" + markup + "</p>");
                }

                if (uploaded.length > 0) {
                  handlers.handleAddAssets(uploaded.map((u) => u.asset));
                }
              }}
            />

            <Button
              type="submit"
              size="sm"
              variant="subtle"
              loading={form.formState.isSubmitting}
            >
              <CreateIcon />
              {t.feed.share}
            </Button>
          </HStack>
        </WStack>
      </styled.form>

      {match(hydratedLink)
        .with(null, () => null)
        .with("loading", () => <Spinner />)
        .otherwise((link: LinkReference) => (
          <Card
            id={link.slug}
            shape="row"
            title={link.title || "(No site title found)"}
            text={link.description || "(No site description found)"}
            image={getAssetURL(link.primary_image?.path)}
            url={link.url}
          />
        ))}
    </CardBox>
  );
}

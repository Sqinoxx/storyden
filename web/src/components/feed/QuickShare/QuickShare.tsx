import { useId } from "react";

import { match } from "ts-pattern";
import { X } from "lucide-react";

import { Asset, LinkReference } from "@/api/openapi-schema";
import { assetUpload } from "@/api/openapi-client/assets";
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
import { CardBox, HStack, WStack, styled } from "@/styled-system/jsx";
import { button } from "@/styled-system/recipes";
import { lstack } from "@/styled-system/patterns";
import { useTranslation } from "@/lib/i18n";
import { getAssetURL } from "@/utils/asset";

import { Props, useQuickShare } from "./useQuickShare";

export function QuickShare(props: Props) {
  const fileInputId = useId();

  const {
    form,
    state: { formRef, hydratedLink, resetKey, uploadedAssets },
    handlers,
  } = useQuickShare(props);
  const t = useTranslation();

  // TODO: Render a prompt to sign up to contribute if not logged in.
  if (!props.initialSession) {
    return null;
  }

  return (
    <CardBox bgColor="bg.default">
      <form
        className={lstack({
          gap: "2",
        })}
        ref={formRef}
        onFocus={handlers.handleFocus}
        onSubmit={handlers.handlePost}
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

        <ComposeField
          control={form.control}
          name="body"
          placeholder={t.feed.quickSharePlaceholder}
          resetKey={resetKey}
        />

        {uploadedAssets.length > 0 && (
          <HStack flexWrap="wrap" gap="2" py="1" px="1">
            {uploadedAssets.map((asset) => (
              <styled.div
                key={asset.id || asset.filename}
                display="inline-flex"
                alignItems="center"
                gap="1"
              >
                <FileAttachmentBadge asset={asset} />
                <styled.button
                  type="button"
                  onClick={() => handlers.handleRemoveAsset(asset.id)}
                  title="Datei entfernen"
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                  w="6"
                  h="6"
                  borderRadius="md"
                  color="fg.muted"
                  _hover={{ bgColor: "bg.muted", color: "fg.default" }}
                  style={{ cursor: "pointer", transition: "all 0.15s ease" }}
                >
                  <X size={14} />
                </styled.button>
              </styled.div>
            ))}
          </HStack>
        )}

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
            <label
              className={button({ size: "sm", variant: "ghost" })}
              htmlFor={fileInputId}
              title={t.editor.uploadFile}
            >
              📎 {t.editor.uploadFile}
            </label>
            <styled.input
              id={fileInputId}
              type="file"
              multiple
              display="none"
              accept="*"
              onChange={async (e) => {
                const files = Array.from(e.currentTarget.files ?? []);
                if (files.length === 0) return;

                const newAssets: Asset[] = [];
                for (const file of files) {
                  try {
                    const asset = await assetUpload(
                      file,
                      { filename: file.name },
                    );
                    newAssets.push(asset);

                    const url = getAssetURL(asset.path);
                    const isImage = /^image\//i.test(file.type);
                    const insertion = isImage
                      ? `<img src="${url}" alt="${file.name}" />`
                      : `<a href="${url}" data-type="file-attachment" data-filename="${file.name}" download="${file.name}">${file.name}</a>`;

                    const current = form.getValues("body") ?? "";
                    form.setValue("body", current + "<p>" + insertion + "</p>");
                  } catch {
                    // ignore upload errors silently
                  }
                }
                if (newAssets.length > 0) {
                  handlers.handleAddAssets(newAssets);
                }
                e.target.value = "";
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
      </form>

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

import { useId } from "react";

import { match } from "ts-pattern";

import { LinkReference } from "@/api/openapi-schema";
import { assetUpload } from "@/api/openapi-client/assets";
import { CategorySelect } from "@/components/category/CategorySelect/CategorySelect";
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
    state: { formRef, hydratedLink, resetKey },
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

        <WStack
          w="full"
          justifyContent={
            props.showCategorySelect ? "space-between" : "flex-end"
          }
        >
          {props.showCategorySelect && (
            <HStack alignItems="center">
              <CategorySelect control={form.control} name="category" />

              <FormErrorText>
                {form.formState.errors["category"]?.message}
              </FormErrorText>
            </HStack>
          )}

          <HStack gap="2" ml={props.showCategorySelect ? "auto" : undefined}>
            <label
              className={button({ size: "sm", variant: "ghost" })}
              htmlFor={fileInputId}
              title="Datei hochladen"
            >
              📎
            </label>
            <styled.input
              id={fileInputId}
              type="file"
              multiple
              display="none"
              accept="image/*,application/pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.zip"
              onChange={async (e) => {
                const files = Array.from(e.currentTarget.files ?? []);
                if (files.length === 0) return;

                for (const file of files) {
                  try {
                    const asset = await assetUpload(
                      file,
                      { filename: file.name },
                    );
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

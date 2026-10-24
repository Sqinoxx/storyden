"use client";

import {
  FileUploadFileAcceptDetails,
  FileUploadFileRejectDetails,
} from "@ark-ui/react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import { assetUpload } from "@/api/openapi-client/assets";
import { Button } from "@/components/ui/button";
import * as FileUpload from "@/components/ui/file-upload";
import { MediaAddIcon } from "@/components/ui/icons/Media";
import { LogoVariant, getLogoURLs } from "@/lib/settings/logo";
import { useSettingsMutation } from "@/lib/settings/mutation";
import { useSettings } from "@/lib/settings/settings-client";
import { css } from "@/styled-system/css";
import { Box, HStack, LStack, Stack, styled } from "@/styled-system/jsx";

const ACCEPTED_LOGO_MIMES = ["image/png", "image/jpeg", "image/webp"];

const VARIANTS: { variant: LogoVariant; label: string }[] = [
  { variant: "light", label: "Light mode" },
  { variant: "dark", label: "Dark mode" },
];

export function LogoEditor() {
  const result = useSettings();
  const { updateSettings, revalidate } = useSettingsMutation();

  if (!result.ready) {
    return null;
  }

  const { settings } = result;
  const logo = settings.metadata.logo;
  const urls = getLogoURLs(logo);

  async function save(variant: LogoVariant, path: string) {
    await updateSettings({
      metadata: {
        ...settings.metadata,
        logo: {
          ...logo,
          [variant]: path,
        },
      },
    });
  }

  function handleAccept(variant: LogoVariant) {
    return async ({ files }: FileUploadFileAcceptDetails) => {
      const file = files[0];
      if (!file) {
        return;
      }

      await handle(
        async () => {
          const asset = await assetUpload(file, { filename: file.name });
          await save(variant, asset.path);
        },
        {
          promiseToast: {
            loading: "Uploading logo...",
            success: "Logo saved",
          },
          cleanup: async () => {
            await revalidate();
          },
        },
      );
    };
  }

  function handleReject({ files }: FileUploadFileRejectDetails) {
    if (files.length === 0) {
      return;
    }

    toast.error("Logo must be a PNG, JPEG or WebP image.");
  }

  async function handleReset(variant: LogoVariant) {
    await handle(
      async () => {
        await save(variant, "");
      },
      {
        promiseToast: {
          loading: "Resetting logo...",
          success: "Logo reset",
        },
        cleanup: async () => {
          await revalidate();
        },
      },
    );
  }

  return (
    <Stack
      w="full"
      gap="4"
      direction={{
        base: "column",
        md: "row",
      }}
    >
      {VARIANTS.map(({ variant, label }) => (
        <LStack key={variant} gap="2" flex="1">
          <styled.span fontSize="sm" fontWeight="medium">
            {label}
          </styled.span>

          <Box
            w="full"
            p="4"
            borderRadius="md"
            borderWidth="thin"
            borderColor="border.subtle"
            bg={variant === "light" ? "white" : "black"}
          >
            {urls[variant] ? (
              <img
                src={urls[variant]}
                alt={`${label} logo`}
                className={css({
                  h: "16",
                  w: "auto",
                  maxW: "full",
                  objectFit: "contain",
                })}
              />
            ) : (
              <styled.span
                display="flex"
                h="16"
                alignItems="center"
                fontSize="sm"
                color="fg.subtle"
              >
                No logo
              </styled.span>
            )}
          </Box>

          <HStack>
            <FileUpload.Root
              w="min"
              maxFiles={1}
              accept={ACCEPTED_LOGO_MIMES}
              onFileAccept={handleAccept(variant)}
              onFileReject={handleReject}
            >
              <FileUpload.Trigger w="min" asChild>
                <Button type="button" size="xs" variant="outline">
                  <MediaAddIcon /> Upload
                </Button>
              </FileUpload.Trigger>
              <FileUpload.HiddenInput />
            </FileUpload.Root>

            {logo[variant] && (
              <Button
                type="button"
                size="xs"
                variant="ghost"
                onClick={() => handleReset(variant)}
              >
                Reset
              </Button>
            )}
          </HStack>
        </LStack>
      ))}
    </Stack>
  );
}

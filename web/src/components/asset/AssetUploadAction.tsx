import {
  FileUploadFileAcceptDetails,
  FileUploadFileRejectDetails,
} from "@ark-ui/react";
import { PropsWithChildren } from "react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import { assetUpload } from "@/api/openapi-client/assets";
import { Asset, AssetID } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import * as FileUpload from "@/components/ui/file-upload";
import { MediaAddIcon, MediaIcon } from "@/components/ui/icons/Media";
import { useTranslation } from "@/lib/i18n";
import { useMaxUploadSizeBytes } from "@/lib/settings/uploads";
import { ButtonVariantProps, button } from "@/styled-system/recipes";
import { getExtensionsForMimeTypes } from "@/utils/mime-types";

type AssetUploadActionProps = {
  parentAssetID?: AssetID;
  operation: "add" | "update";
  onFinish: (a: Asset) => Promise<void>;
  hideLabel?: boolean;
};

type Props = AssetUploadActionProps & ButtonVariantProps & FileUpload.RootProps;

export function AssetUploadAction({
  children,
  parentAssetID,
  operation,
  onFinish,
  hideLabel,
  ...props
}: PropsWithChildren<Props>) {
  const t = useTranslation();
  const [buttonVariantProps, fileUploadProps] = button.splitVariantProps(props);

  const acceptedMIMEs = getMIMEs(props.accept);
  const maxUploadSizeBytes = useMaxUploadSizeBytes();

  async function handleFile({ files }: FileUploadFileAcceptDetails) {
    await handle(async () => {
      // NOTE: For some reason (Zag bug?) this is called for rejected files too.
      const file = files[0];
      if (!file) {
        return;
      }

      if (file.size > maxUploadSizeBytes) {
        throw new Error(
          t.upload.tooLarge.replace(
            "{size}",
            String(Math.floor(maxUploadSizeBytes / 1024 / 1024)),
          ),
        );
      }

      const asset = await assetUpload(file, {
        filename: file.name,
        parent_asset_id: parentAssetID,
      });

      // Awaited so a failure in the caller's follow-up work (a draft save, a
      // revalidation) is reported by the enclosing handle instead of escaping
      // as an unhandled rejection.
      await onFinish({ ...asset, filename: file.name });
    });
  }

  async function handleFileReject({ files }: FileUploadFileRejectDetails) {
    if (files.length === 0) {
      return;
    }

    const file = files[0];
    if (!file) {
      return;
    }

    const accepted = getExtensionsForMimeTypes(acceptedMIMEs);

    const acceptedList = accepted.map((e) => `.${e}`).join(", ");

    // Vast majority of the time, there will only be one error, but join anyway.
    const errorMessage = file.errors
      .map((error) => {
        switch (error) {
          case "FILE_INVALID":
            return t.upload.invalidFile;
          case "FILE_TOO_LARGE":
            return t.upload.tooLarge.replace(
              "{size}",
              String(Math.floor(maxUploadSizeBytes / 1024 / 1024)),
            );
          case "FILE_INVALID_TYPE":
            return t.upload.invalidType.replace("{types}", acceptedList);
          case "FILE_TOO_SMALL":
            return t.upload.tooSmall;
          case "TOO_MANY_FILES":
            return t.upload.tooMany;
          default:
            return t.upload.readError;
        }
      })
      .join(", ");

    toast.error(errorMessage);
  }

  return (
    <FileUpload.Root
      w="min"
      maxFiles={1}
      maxFileSize={maxUploadSizeBytes}
      onFileAccept={handleFile}
      onFileReject={handleFileReject}
      {...fileUploadProps}
    >
      <FileUpload.Trigger asChild>
        {children || (
          <Button
            type="button"
            size="xs"
            variant="outline"
            {...buttonVariantProps}
          >
            {operation === "add" ? (
              <>
                <MediaAddIcon />
                {hideLabel ? "" : t.library.addCover}
              </>
            ) : (
              <>
                <MediaIcon /> {hideLabel ? "" : t.library.replaceCover}
              </>
            )}
          </Button>
        )}
      </FileUpload.Trigger>
      <FileUpload.HiddenInput data-testid="input" />
    </FileUpload.Root>
  );
}

// NOTE: For some reason, Ark UI's prop type for "accept" also includes a record
// type (not sure what the use-case is) so, we need to convert it into an array.
function getMIMEs(
  accept: Record<string, string[]> | string | string[] | undefined,
): string[] {
  if (!accept) {
    return [];
  }

  if (typeof accept === "string") {
    return [accept];
  }

  if (Array.isArray(accept)) {
    return accept;
  }

  const mimes = Object.keys(accept);

  return mimes;
}

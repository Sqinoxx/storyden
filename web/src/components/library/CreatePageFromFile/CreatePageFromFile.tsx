import { FileUploadFileAcceptDetails } from "@ark-ui/react";
import { Upload } from "lucide-react";
import { useState } from "react";

import { handle } from "@/api/client";
import { assetUpload } from "@/api/openapi-client/assets";
import { nodeAddAsset, nodeCreate } from "@/api/openapi-client/nodes";
import { ButtonProps } from "@/components/ui/button";
import * as FileUpload from "@/components/ui/file-upload";
import { IconButton } from "@/components/ui/icon-button";
import { Item } from "@/components/ui/menu";
import { useLibraryMutation } from "@/lib/library/library";
import { styled } from "@/styled-system/jsx";

export type CreatePageFromFileProps = ButtonProps & {
  parentSlug?: string;
  hideLabel?: boolean;
  onComplete?: () => void;
};

export const CreatePageFromFileID = "create-page-from-file";
export const CreatePageFromFileLabel = "Upload file";
export const CreatePageFromFileIcon = <Upload size={16} />;

export function CreatePageFromFileAction({
  parentSlug,
  hideLabel,
  onComplete,
  ...props
}: CreatePageFromFileProps) {
  const { revalidate } = useLibraryMutation();
  const [isUploading, setIsUploading] = useState(false);

  async function handleFileAccept({ files }: FileUploadFileAcceptDetails) {
    const file = files[0];
    if (!file) return;

    setIsUploading(true);
    await handle(
      async () => {
        const isImage = file.type.startsWith("image/");
        const uploadedAsset = await assetUpload(file, {
          filename: file.name,
        });

        const createdNode = await nodeCreate({
          name: file.name,
          parent: parentSlug,
          primary_image_asset_id: isImage ? uploadedAsset.id : undefined,
        });

        if (createdNode && uploadedAsset) {
          await nodeAddAsset(createdNode.id, uploadedAsset.id);
        }
      },
      {
        promiseToast: {
          loading: "Uploading file & creating page...",
          success: "Page created with file!",
        },
        cleanup: async () => {
          setIsUploading(false);
          await revalidate();
          onComplete?.();
        },
      },
    );
  }

  return (
    <FileUpload.Root
      w="min"
      maxFiles={1}
      onFileAccept={handleFileAccept}
      disabled={isUploading}
    >
      <FileUpload.Trigger asChild>
        <IconButton
          type="button"
          size="xs"
          variant="subtle"
          px={hideLabel ? "0" : "1"}
          disabled={isUploading}
          {...props}
        >
          {CreatePageFromFileIcon}
          {!hideLabel && <span>{CreatePageFromFileLabel}</span>}
        </IconButton>
      </FileUpload.Trigger>
      <FileUpload.HiddenInput />
    </FileUpload.Root>
  );
}

export function CreatePageFromFileMenuItem({ hideLabel }: CreatePageFromFileProps) {
  return (
    <Item value={CreatePageFromFileID}>
      {CreatePageFromFileIcon}
      {!hideLabel && (
        <>
          &nbsp;<span>{CreatePageFromFileLabel}</span>
        </>
      )}
    </Item>
  );
}

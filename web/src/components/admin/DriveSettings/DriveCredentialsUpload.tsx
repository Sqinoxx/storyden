import { FileUploadFileChangeDetails } from "@ark-ui/react";
import { FileUploadFileError as FileError } from "@ark-ui/react/file-upload";

import { Button } from "@/components/ui/button";
import * as FileUpload from "@/components/ui/file-upload";
import { AddIcon } from "@/components/ui/icons/Add";
import { VStack, styled } from "@/styled-system/jsx";

type Props = {
  disabled?: boolean;
  buttonLabel?: string;
  onFileChange: (file: File) => void;
  onError?: (error: string) => void;
};

export function DriveCredentialsUpload({
  disabled,
  buttonLabel = "Select File",
  onFileChange,
  onError,
}: Props) {
  function handleFileChange(details: FileUploadFileChangeDetails) {
    const file = details.acceptedFiles[0];
    if (file) {
      onFileChange(file);
    }

    const rejectionError = mapRejectionToError(details);
    if (rejectionError) {
      onError?.(rejectionError);
    }
  }

  return (
    <FileUpload.Root
      maxFiles={1}
      accept="application/json,.json"
      maxFileSize={1024 * 1024}
      onFileChange={handleFileChange}
      disabled={disabled}
    >
      <FileUpload.Dropzone minHeight="32">
        <VStack>
          <VStack gap="1" textAlign="center">
            <styled.p fontWeight="medium">
              Drop the service account key here or click to browse
            </styled.p>
            <styled.p fontSize="sm" color="fg.muted">
              JSON key file, max 1MB
            </styled.p>
          </VStack>
          <FileUpload.Trigger asChild>
            <Button variant="outline" disabled={disabled} size="sm">
              <AddIcon />
              {buttonLabel}
            </Button>
          </FileUpload.Trigger>
        </VStack>
      </FileUpload.Dropzone>

      <FileUpload.HiddenInput />
    </FileUpload.Root>
  );
}

function mapRejectionToError(
  details: FileUploadFileChangeDetails,
): string | undefined {
  const file = details.rejectedFiles[0];
  if (!file) {
    return undefined;
  }

  const messages = file.errors.map(mapFileError).filter(Boolean);
  if (messages.length === 0) {
    return undefined;
  }

  return messages.join(", ");
}

function mapFileError(error: FileError): string {
  switch (error) {
    case "FILE_INVALID":
      return "Invalid file.";
    case "FILE_TOO_LARGE":
      return "Key file is too large. Maximum size is 1MB.";
    case "FILE_INVALID_TYPE":
      return "File must be a JSON key file.";
    case "FILE_TOO_SMALL":
      return "File is too small.";
    case "TOO_MANY_FILES":
      return "Only one key file can be uploaded at a time.";
    case "FILE_EXISTS":
      return "A file with this name has already been selected.";
    default:
      return "An unexpected error occurred while reading the file.";
  }
}

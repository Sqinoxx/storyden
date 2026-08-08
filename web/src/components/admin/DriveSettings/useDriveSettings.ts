import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import {
  adminDriveCredentialsDelete,
  adminDriveCredentialsUpload,
  adminDriveFolderCreate,
  adminDriveFolderDelete,
  adminDriveFolderUpdate,
  useAdminDriveCredentialsGet,
  useAdminDriveFolderList,
} from "@/api/openapi-client/drive";
import { DriveFolder, DriveVisibility } from "@/api/openapi-schema";
import { deriveError } from "@/utils/error";

export const FormSchema = z.object({
  name: z.string().min(1, "Give the folder a name members will recognise."),
  link: z.string().min(1, "Paste the Google Drive folder link."),
  description: z.string(),
  visibility: z.enum(["public", "member", "admin"]),
});

export type Form = z.infer<typeof FormSchema>;

const DEFAULTS: Form = {
  name: "",
  link: "",
  description: "",
  visibility: "member",
};

export function useDriveSettings() {
  const { data, error, mutate } = useAdminDriveFolderList();
  const {
    data: credentials,
    error: credentialsError,
    mutate: mutateCredentials,
  } = useAdminDriveCredentialsGet();
  const [isUploadingCredentials, setIsUploadingCredentials] = useState(false);
  const [credentialsUploadError, setCredentialsUploadError] = useState<
    string | null
  >(null);

  async function handleCredentialsUpload(file: File) {
    setIsUploadingCredentials(true);
    setCredentialsUploadError(null);

    await handle(
      async () => {
        await adminDriveCredentialsUpload(file);
        await mutateCredentials();
        await mutate();
      },
      {
        promiseToast: {
          loading: "Uploading service account key...",
          success: "Service account key uploaded",
        },
        async onError(err) {
          setCredentialsUploadError(deriveError(err));
        },
        cleanup: async () => {
          setIsUploadingCredentials(false);
        },
      },
    );
  }

  async function handleCredentialsRemove() {
    await handle(
      async () => {
        await adminDriveCredentialsDelete();
        await mutateCredentials();
        await mutate();
      },
      {
        promiseToast: {
          loading: "Removing service account key...",
          success: "Service account key removed",
        },
      },
    );
  }

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: DEFAULTS,
  });

  const onSubmit = form.handleSubmit(async (values) => {
    await handle(
      async () => {
        await adminDriveFolderCreate({
          name: values.name,
          link: values.link,
          description: values.description || undefined,
          visibility: values.visibility,
        });

        form.reset(DEFAULTS);
      },
      {
        promiseToast: {
          loading: "Adding folder...",
          success: "Folder added",
        },
        cleanup: async () => {
          await mutate();
        },
      },
    );
  });

  async function handleVisibilityChange(
    folder: DriveFolder,
    visibility: DriveVisibility,
  ) {
    await handle(
      async () => {
        await adminDriveFolderUpdate(folder.id, { visibility });
      },
      {
        promiseToast: {
          loading: "Updating folder...",
          success: "Folder updated",
        },
        cleanup: async () => {
          await mutate();
        },
      },
    );
  }

  async function handleDelete(folder: DriveFolder) {
    await handle(
      async () => {
        await adminDriveFolderDelete(folder.id);
      },
      {
        promiseToast: {
          loading: "Removing folder...",
          success: "Folder removed",
        },
        cleanup: async () => {
          await mutate();
        },
      },
    );
  }

  if (!data || !credentials) {
    return {
      ready: false as const,
      error: error ?? credentialsError,
    };
  }

  return {
    ready: true as const,
    folders: data.folders,
    configured: data.configured,
    credentials,
    isUploadingCredentials,
    credentialsUploadError,
    handleCredentialsUpload,
    handleCredentialsRemove,
    form,
    onSubmit,
    handleVisibilityChange,
    handleDelete,
  };
}

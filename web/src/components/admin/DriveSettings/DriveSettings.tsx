"use client";

import { createListCollection } from "@ark-ui/react";

import { DriveCredentialsStatus, DriveFolder, DriveVisibility } from "@/api/openapi-schema";
import { DeleteWithConfirmationButton } from "@/components/site/DeleteConfirmationButton";
import { Unready } from "@/components/site/Unready";
import { Admonition } from "@/components/ui/admonition";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { SelectField } from "@/components/ui/form/SelectField";
import { Heading } from "@/components/ui/heading";
import { SelectIcon } from "@/components/ui/icons/Select";
import { Input } from "@/components/ui/input";
import * as Select from "@/components/ui/select";
import * as Table from "@/components/ui/table";
import { CardBox, HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { DriveCredentialsUpload } from "./DriveCredentialsUpload";
import { useDriveSettings } from "./useDriveSettings";

const VISIBILITY_COLLECTION = createListCollection({
  items: [
    { label: "Everyone", value: "public" },
    { label: "Members only", value: "member" },
    { label: "Admins only", value: "admin" },
  ],
});

export function DriveSettings() {
  const state = useDriveSettings();

  if (!state.ready) return <Unready error={state.error} />;

  const {
    folders,
    credentials,
    isUploadingCredentials,
    credentialsUploadError,
    handleCredentialsUpload,
    handleCredentialsRemove,
    form,
    onSubmit,
    handleVisibilityChange,
    handleDelete,
  } = state;

  return (
    <LStack gap="4">
      <DriveCredentialsCard
        credentials={credentials}
        isUploading={isUploadingCredentials}
        uploadError={credentialsUploadError}
        onUpload={handleCredentialsUpload}
        onRemove={handleCredentialsRemove}
      />

      <Admonition
        value={!credentials.configured}
        kind="neutral"
        title="Google Drive is not configured"
      >
        <styled.p fontSize="sm">
          Upload a service account key above, or set{" "}
          <styled.code>GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON</styled.code> and
          restart the server. Until then, folders added here cannot be read.
        </styled.p>
      </Admonition>

      <CardBox className={lstack()}>
        <Heading size="md">Shared folders</Heading>

        {folders.length === 0 ? (
          <styled.p color="fg.muted">
            No folders yet. Add one below to make it browsable.
          </styled.p>
        ) : (
          <styled.div w="full" overflowX="auto">
            <Table.Root size="sm" variant="dense">
              <Table.Head>
                <Table.Row>
                  <Table.Header>Name</Table.Header>
                  <Table.Header>Drive folder</Table.Header>
                  <Table.Header>Visible to</Table.Header>
                  <Table.Header />
                </Table.Row>
              </Table.Head>

              <Table.Body>
                {folders.map((folder) => (
                  <Table.Row key={folder.id}>
                    <Table.Cell>{folder.name}</Table.Cell>

                    <Table.Cell color="fg.muted">
                      <styled.code fontSize="xs">
                        {folder.drive_folder_id}
                      </styled.code>
                    </Table.Cell>

                    <Table.Cell>
                      <VisibilitySelect
                        folder={folder}
                        onChange={handleVisibilityChange}
                      />
                    </Table.Cell>

                    <Table.Cell>
                      <HStack justify="end">
                        <DeleteWithConfirmationButton
                          size="xs"
                          variant="ghost"
                          onDelete={() => handleDelete(folder)}
                        />
                      </HStack>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table.Root>
          </styled.div>
        )}
      </CardBox>

      <styled.form
        width="full"
        display="flex"
        flexDirection="column"
        gap="4"
        onSubmit={onSubmit}
      >
        <CardBox className={lstack()}>
          <WStack>
            <Heading size="md">Add a folder</Heading>
            <Button type="submit" loading={form.formState.isSubmitting}>
              Add
            </Button>
          </WStack>

          <FormControl>
            <FormLabel>Name</FormLabel>
            <Input aria-label="Name" {...form.register("name")} />
            <FormErrorText>{form.formState.errors.name?.message}</FormErrorText>
          </FormControl>

          <FormControl>
            <FormLabel>Google Drive folder link</FormLabel>
            <Input
              aria-label="Google Drive folder link"
              placeholder="https://drive.google.com/drive/folders/..."
              {...form.register("link")}
            />
            <FormHelperText>
              Share the folder with the service account&apos;s email address in
              Google Drive first, otherwise it cannot be read.
            </FormHelperText>
            <FormErrorText>{form.formState.errors.link?.message}</FormErrorText>
          </FormControl>

          <FormControl>
            <FormLabel>Description</FormLabel>
            <Input aria-label="Description" {...form.register("description")} />
          </FormControl>

          <FormControl>
            <FormLabel>Visible to</FormLabel>
            <SelectField
              control={form.control}
              name="visibility"
              collection={VISIBILITY_COLLECTION}
              placeholder="Members only"
            />
          </FormControl>
        </CardBox>
      </styled.form>
    </LStack>
  );
}

function DriveCredentialsCard({
  credentials,
  isUploading,
  uploadError,
  onUpload,
  onRemove,
}: {
  credentials: DriveCredentialsStatus;
  isUploading: boolean;
  uploadError: string | null;
  onUpload: (file: File) => void;
  onRemove: () => Promise<void>;
}) {
  return (
    <CardBox className={lstack()}>
      <Heading size="md">Google Drive access</Heading>

      {credentials.configured ? (
        <LStack gap="2">
          <WStack alignItems="start">
            <LStack gap="0">
              <styled.p fontSize="sm">
                {credentials.source === "upload"
                  ? "Using an uploaded service account key."
                  : "Using the service account key from environment configuration."}
              </styled.p>
              {credentials.service_account_email && (
                <styled.p fontSize="sm" color="fg.muted">
                  Share Drive folders with{" "}
                  <styled.code>
                    {credentials.service_account_email}
                  </styled.code>{" "}
                  to make them reachable.
                </styled.p>
              )}
            </LStack>

            {credentials.source === "upload" && (
              <DeleteWithConfirmationButton
                size="xs"
                variant="outline"
                onDelete={onRemove}
              >
                Remove key
              </DeleteWithConfirmationButton>
            )}
          </WStack>

          {credentials.source === "environment" && (
            <styled.p fontSize="sm" color="fg.muted">
              Upload a key below to override the environment configuration.
            </styled.p>
          )}
        </LStack>
      ) : (
        <styled.p fontSize="sm" color="fg.muted">
          Upload the JSON key for a Google service account to let this
          installation read shared Drive folders.
        </styled.p>
      )}

      <DriveCredentialsUpload
        disabled={isUploading}
        buttonLabel={isUploading ? "Uploading..." : "Select File"}
        onFileChange={onUpload}
      />

      {uploadError && <FormErrorText>{uploadError}</FormErrorText>}
    </CardBox>
  );
}

function VisibilitySelect({
  folder,
  onChange,
}: {
  folder: DriveFolder;
  onChange: (folder: DriveFolder, visibility: DriveVisibility) => Promise<void>;
}) {
  return (
    <Select.Root
      size="sm"
      minW="max"
      positioning={{ sameWidth: false }}
      collection={VISIBILITY_COLLECTION}
      value={[folder.visibility]}
      onValueChange={({ value }) => {
        const [next] = value;
        if (next && next !== folder.visibility) {
          onChange(folder, next as DriveVisibility);
        }
      }}
    >
      <Select.Control>
        <Select.Trigger aria-label={`Visibility for ${folder.name}`}>
          <Select.ValueText />
          <SelectIcon />
        </Select.Trigger>
      </Select.Control>
      <Select.Positioner>
        <Select.Content>
          {VISIBILITY_COLLECTION.items.map((item) => (
            <Select.Item key={item.value} item={item}>
              <Select.ItemText>{item.label}</Select.ItemText>
            </Select.Item>
          ))}
        </Select.Content>
      </Select.Positioner>
    </Select.Root>
  );
}

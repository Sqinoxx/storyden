"use client";

import { Download, Eye, File, Folder } from "lucide-react";
import Link from "next/link";
import { useState } from "react";

import { DriveEntry } from "@/api/openapi-schema";
import { FilePreviewModal } from "@/components/post/FilePreviewModal";
import { IconButton } from "@/components/ui/icon-button";
import * as Table from "@/components/ui/table";
import { HStack, styled } from "@/styled-system/jsx";
import { downloadAsset } from "@/utils/asset";

import { driveFileSize, driveKindLabel, driveModifiedAt } from "./format";
import { getDriveFileURL, getDriveFolderHref } from "./url";

type Props = {
  folderID: string;
  entries: DriveEntry[];
};

export function DriveContentsTable({ folderID, entries }: Props) {
  const [previewing, setPreviewing] = useState<DriveEntry | undefined>();

  async function handleDownload(entry: DriveEntry) {
    await downloadAsset(
      getDriveFileURL(folderID, entry.id, "attachment"),
      entry.name,
    );
  }

  return (
    <>
      <styled.div w="full" overflowX="auto">
        <Table.Root size="sm" variant="dense">
          <Table.Head>
            <Table.Row>
              <Table.Header>Name</Table.Header>
              <Table.Header>Type</Table.Header>
              <Table.Header>Size</Table.Header>
              <Table.Header>Modified</Table.Header>
              <Table.Header />
            </Table.Row>
          </Table.Head>

          <Table.Body>
            {entries.map((entry) => (
              <Table.Row key={entry.id}>
                <Table.Cell>
                  <HStack gap="2" minW="0">
                    {entry.is_folder ? (
                      <Folder size={16} />
                    ) : (
                      <File size={16} />
                    )}

                    {entry.is_folder ? (
                      <styled.span truncate>
                        <Link href={getDriveFolderHref(folderID, entry.id)}>
                          {entry.name}
                        </Link>
                      </styled.span>
                    ) : entry.previewable ? (
                      <styled.button
                        type="button"
                        truncate
                        textAlign="left"
                        cursor="pointer"
                        _hover={{ textDecoration: "underline" }}
                        onClick={() => setPreviewing(entry)}
                      >
                        {entry.name}
                      </styled.button>
                    ) : (
                      <styled.span truncate>{entry.name}</styled.span>
                    )}
                  </HStack>
                </Table.Cell>

                <Table.Cell color="fg.muted">
                  {driveKindLabel(entry.mime_type, entry.is_folder)}
                </Table.Cell>

                <Table.Cell color="fg.muted">
                  {driveFileSize(entry.size)}
                </Table.Cell>

                <Table.Cell color="fg.muted">
                  {driveModifiedAt(entry.modified_at)}
                </Table.Cell>

                <Table.Cell>
                  {!entry.is_folder && (
                    <HStack gap="1" justify="end">
                      {entry.previewable && (
                        <IconButton
                          type="button"
                          size="xs"
                          variant="ghost"
                          aria-label={`Preview ${entry.name}`}
                          onClick={() => setPreviewing(entry)}
                        >
                          <Eye size={16} />
                        </IconButton>
                      )}

                      <IconButton
                        type="button"
                        size="xs"
                        variant="ghost"
                        aria-label={`Download ${entry.name}`}
                        onClick={() => handleDownload(entry)}
                      >
                        <Download size={16} />
                      </IconButton>
                    </HStack>
                  )}
                </Table.Cell>
              </Table.Row>
            ))}
          </Table.Body>
        </Table.Root>
      </styled.div>

      {previewing && (
        <FilePreviewModal
          url={getDriveFileURL(folderID, previewing.id, "inline")}
          displayName={previewing.name}
          mimeType={previewing.mime_type}
          onClose={() => setPreviewing(undefined)}
          onDownload={() => handleDownload(previewing)}
        />
      )}
    </>
  );
}

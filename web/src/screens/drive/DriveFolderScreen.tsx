"use client";

import { Folder } from "lucide-react";

import { DriveContentsTable } from "@/components/drive/DriveContentsTable";
import { getDriveFolderHref } from "@/components/drive/url";
import { EmptyState } from "@/components/site/EmptyState";
import { Unready } from "@/components/site/Unready";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { VStack } from "@/styled-system/jsx";

import { Props, useDriveFolderScreen } from "./useDriveFolderScreen";

export function DriveFolderScreen(props: Props) {
  const { ready, data, error } = useDriveFolderScreen(props);

  if (!ready) return <Unready error={error} />;

  // The first crumb is the registered folder itself, which the index link
  // already covers.
  const [, ...descendants] = data.breadcrumbs;

  return (
    <VStack gap="4" alignItems="start" w="full">
      <Breadcrumbs
        index={{ label: "Drive", href: "/drive" }}
        crumbs={[
          {
            label: data.folder.name,
            href: getDriveFolderHref(data.folder.id),
          },
          ...descendants.map((crumb) => ({
            label: crumb.name,
            href: getDriveFolderHref(data.folder.id, crumb.id),
          })),
        ]}
      />

      {data.entries.length === 0 ? (
        <EmptyState icon={<Folder />} hideContributionLabel w="full">
          <p>This folder is empty.</p>
        </EmptyState>
      ) : (
        <DriveContentsTable folderID={data.folder.id} entries={data.entries} />
      )}
    </VStack>
  );
}

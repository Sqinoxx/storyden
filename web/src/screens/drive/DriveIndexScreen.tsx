"use client";

import { Folder } from "lucide-react";

import { DriveFolderGrid } from "@/components/drive/DriveFolderGrid";
import { EmptyState } from "@/components/site/EmptyState";
import { Unready } from "@/components/site/Unready";
import { Heading } from "@/components/ui/heading";
import { useTranslation } from "@/lib/i18n";
import { VStack } from "@/styled-system/jsx";

import { Props, useDriveIndexScreen } from "./useDriveIndexScreen";

export function DriveIndexScreen(props: Props) {
  const t = useTranslation();
  const { ready, data, error } = useDriveIndexScreen(props);

  if (!ready) return <Unready error={error} />;

  return (
    <VStack gap="4" alignItems="start" w="full">
      <Heading size="lg">{t.nav.drive}</Heading>

      {data.folders.length === 0 ? (
        <EmptyState
          icon={<Folder />}
          hideContributionLabel
          w="full"
          unauthenticatedLabel=""
        >
          <p>{t.drive.noFolders}</p>
        </EmptyState>
      ) : (
        <DriveFolderGrid folders={data.folders} />
      )}
    </VStack>
  );
}

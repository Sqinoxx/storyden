import { Folder, Lock, Users } from "lucide-react";
import Link from "next/link";

import { DriveFolder, DriveVisibility } from "@/api/openapi-schema";
import { Heading } from "@/components/ui/heading";
import { CardGrid } from "@/components/ui/rich-card";
import { CardBox, HStack, VStack, styled } from "@/styled-system/jsx";

import { getDriveFolderHref } from "./url";

type Props = {
  folders: DriveFolder[];
};

export function DriveFolderGrid({ folders }: Props) {
  return (
    <CardGrid>
      {folders.map((folder) => (
        <CardBox key={folder.id} p="3">
          <Link href={getDriveFolderHref(folder.id)}>
            <VStack alignItems="start" gap="1">
              <HStack gap="2" w="full" minW="0">
                <Folder size={18} />

                <Heading size="sm" truncate>
                  {folder.name}
                </Heading>

                <VisibilityHint visibility={folder.visibility} />
              </HStack>

              {folder.description && (
                <styled.p color="fg.muted" fontSize="sm" lineClamp="2">
                  {folder.description}
                </styled.p>
              )}
            </VStack>
          </Link>
        </CardBox>
      ))}
    </CardGrid>
  );
}

/**
 * Only shown for folders not everyone can see, so members understand why a link
 * they share may not work for the person they send it to.
 */
function VisibilityHint({ visibility }: { visibility: DriveVisibility }) {
  if (visibility === "public") return null;

  const label = visibility === "admin" ? "Admins only" : "Members only";
  const Icon = visibility === "admin" ? Lock : Users;

  return (
    <styled.span
      color="fg.subtle"
      title={label}
      aria-label={label}
      flexShrink="0"
    >
      <Icon size={14} />
    </styled.span>
  );
}

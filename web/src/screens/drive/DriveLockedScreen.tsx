"use client";

import { Folder } from "lucide-react";

import { DriveFolder } from "@/api/openapi-schema";
import { LockedContentNotice } from "@/components/access/AccessPassCard";
import { Heading } from "@/components/ui/heading";
import { HStack, LStack, styled } from "@/styled-system/jsx";

type Props = {
  folder: DriveFolder;
};

export function DriveLockedScreen({ folder }: Props) {
  const passes = folder.access?.passes ?? [];

  return (
    <LStack gap="4" w="full">
      <HStack gap="2" alignItems="start" minW="0">
        <styled.span flexShrink="0" pt="1">
          <Folder size={20} />
        </styled.span>
        <Heading size="lg" overflowWrap="anywhere">
          {folder.name}
        </Heading>
      </HStack>
      {folder.description && (
        <styled.p color="fg.muted">{folder.description}</styled.p>
      )}
      <LockedContentNotice
        passIDs={passes.map((p) => p.id)}
        fallbackNames={passes.map((p) => p.name)}
      />
    </LStack>
  );
}

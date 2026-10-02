"use client";

import { ProfileReference } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { MediaAddIcon } from "@/components/ui/icons/Media";
import { useTranslation } from "@/lib/i18n";
import { Box } from "@/styled-system/jsx";

import { EditAvatarTrigger } from "../EditAvatar/EditAvatarModal";

export type Props = {
  profile: ProfileReference;
};

export function MemberAvatarEditable({ profile }: Props) {
  const t = useTranslation();
  return (
    <Box position="absolute" w="full" h="full">
      <EditAvatarTrigger profile={profile} asChild>
        <Button
          type="button"
          aria-label={t.profile.changeAvatar}
          position="absolute"
          top="0"
          left="0"
          w="full"
          h="full"
          borderRadius="full"
          variant="subtle"
          color="bg.default"
          size="2xl"
        >
          <MediaAddIcon />
        </Button>
      </EditAvatarTrigger>
    </Box>
  );
}

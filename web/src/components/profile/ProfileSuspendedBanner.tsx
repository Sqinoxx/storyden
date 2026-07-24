"use client";

import { formatDistanceToNow } from "date-fns";

import { Box, CardBox, Flex, HStack, styled } from "@/styled-system/jsx";

import { BanIcon } from "../ui/icons/BanIcon";
import { useTranslation } from "@/lib/i18n";

type Props = {
  date: Date;
};

export function ProfileSuspendedBanner({ date }: Props) {
  const t = useTranslation();
  return (
    <CardBox
      p="0"
      borderColor="border.error"
      borderWidth="thin"
      borderStyle="dashed"
    >
      <Box bgColor="bg.error" borderTopRadius="md" pl="3" pr="2" py="1">
        <HStack gap="1" color="fg.error" fontSize="xs">
          <BanIcon w="4" />
          <p>{t.profile.suspended}</p>
        </HStack>
      </Box>

      <Flex
        p="3"
        gap="4"
        direction={{ base: "column", md: "row" }}
        alignItems="start"
      >
        <styled.p color="fg.destructive" wordBreak="keep-all">
          {t.profile.memberSuspended}&nbsp;
          <styled.time textWrap="nowrap">
            {formatDistanceToNow(date, {
              addSuffix: true,
            })}
          </styled.time>
        </styled.p>
      </Flex>
    </CardBox>
  );
}

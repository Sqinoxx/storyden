import Image from "next/image";
import { PropsWithChildren } from "react";

import { Fullpage } from "@/layouts/Fullpage";

import { BackAction } from "@/components/site/Action/Back";
import { ImpressumLink } from "@/components/site/ImpressumModal/ImpressumLink";
import { CookieLink } from "@/components/site/CookieNotice/CookieLink";
import { getSettings } from "@/lib/settings/settings-server";
import { css } from "@/styled-system/css";
import { CardBox, HStack, VStack, WStack, styled } from "@/styled-system/jsx";
import { vstack } from "@/styled-system/patterns";
import { getIconURL } from "@/utils/icon";

export const dynamic = "force-dynamic";
export const revalidate = 0;
export const fetchCache = "force-no-store";

export default async function Layout({ children }: PropsWithChildren) {
  const settings = await getSettings();

  const siteName = settings.title;

  return (
    <Fullpage>
      <VStack minH="dvh" py="6" justifyContent="space-between" alignItems="center" w="full">
        <styled.div flexGrow={1} />
        <CardBox className={vstack()} maxW="sm" gap="4" p="4">
          <WStack>
            <BackAction />
          </WStack>
          <VStack>
            <Image
              className={css({ width: "24", borderRadius: "md" })}
              src={getIconURL("512x512")}
              width={512}
              height={512}
              alt={`The ${siteName} logo`}
            />

            <styled.h1 fontWeight="bold" fontSize="lg">
              {siteName}
            </styled.h1>
          </VStack>

          {children}
        </CardBox>
        <styled.div flexGrow={2} />

        <HStack color="fg.subtle" fontSize="xs" flexShrink="0" py="2" gap="2">
          <ImpressumLink />
          <styled.span color="fg.subtle">•</styled.span>
          <CookieLink />
          <styled.span color="fg.subtle">•</styled.span>
          <p>powered by storyden</p>
        </HStack>
      </VStack>
    </Fullpage>
  );
}

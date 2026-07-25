"use client";

import { match } from "ts-pattern";
import { useTranslation } from "@/lib/i18n";

import { LinkCard } from "@/components/library/links/LinkCard";
import { InfoTip } from "@/components/site/InfoTip";
import { Unready } from "@/components/site/Unready";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Center, HStack, LStack, WStack } from "@/styled-system/jsx";

import { useWatch } from "../../store";
import { useEditState } from "../../useEditState";

import { useLibraryPageLinkBlock } from "./useLibraryPageLinkBlock";

export function LibraryPageLinkBlock() {
  const { isDirectEditing } = useEditState();

  const link = useWatch((s) => s.draft.link);

  if (isDirectEditing) {
    return <LibraryPageLinkBlockEditing />;
  }

  if (!link?.url) {
    return null;
  }

  return <LinkCard link={link} />;
}

function LibraryPageLinkBlockEditing() {
  const { data, handlers } = useLibraryPageLinkBlock();
  const t = useTranslation();

  return (
    <LStack gap="0">
      <WStack>
        <Input
          w="full"
          size="sm"
          variant="ghost"
          color="fg.muted"
          placeholder={t.library.externalUrlPlaceholder}
          onChange={handlers.handleInputValueChange}
          value={data.inputValue}
        />

        <HStack>
          <InfoTip title={t.library.importTitle}>
            {t.library.importDescription}
          </InfoTip>
          <Button
            type="button"
            size="xs"
            variant="subtle"
            disabled={!data.resolvedLink}
            loading={data.isImporting}
            onClick={handlers.handleImport}
          >
            {t.library.import}
          </Button>
        </HStack>
      </WStack>

      {match(data.resolvedLink)
        .with(undefined, () => null)
        .with(null, () => (
          <Center w="full" h="24">
            <Unready />
          </Center>
        ))
        .otherwise((link) => (
          <LinkCard link={link} />
        ))}
    </LStack>
  );
}

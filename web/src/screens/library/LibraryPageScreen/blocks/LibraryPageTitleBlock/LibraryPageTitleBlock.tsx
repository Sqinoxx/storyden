import { Download } from "lucide-react";

import { IntelligenceAction } from "@/components/site/Action/Intelligence";
import { Heading } from "@/components/ui/heading";
import { HeadingInput } from "@/components/ui/heading-input";
import { HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";

import { useWatch } from "../../store";
import { useEditState } from "../../useEditState";

import { useLibraryPageTitleBlock } from "./useLibraryPageTitleBlock";

export function LibraryPageTitleBlock() {
  const name = useWatch((s) => s.draft.name);
  const assets = useWatch((s) => s.draft.assets);
  const { editing } = useEditState();

  if (editing) {
    return <LibraryPageTitleBlockEditing />;
  }

  const fileAsset = assets && assets.length > 0 ? assets[0] : undefined;
  const downloadUrl = fileAsset ? getAssetURL(fileAsset.path) : null;

  return (
    <LStack gap="2">
      <Heading fontSize="heading.2" fontWeight="bold">
        {name || "(untitled)"}
      </Heading>

      {downloadUrl && (
        <HStack gap="2">
          <styled.a
            href={downloadUrl}
            download={fileAsset?.filename || name}
            target="_blank"
            rel="noopener noreferrer"
            display="inline-flex"
            alignItems="center"
            gap="2"
            px="3"
            py="1.5"
            borderRadius="md"
            bgColor="bg.subtle"
            borderWidth="thin"
            borderColor="border.subtle"
            fontSize="sm"
            fontWeight="medium"
            _hover={{ bgColor: "bg.muted" }}
          >
            <Download size={16} />
            <span>{fileAsset?.filename || "Datei herunterladen"}</span>
          </styled.a>
        </HStack>
      )}
    </LStack>
  );
}

function LibraryPageTitleBlockEditing() {
  const {
    defaultValue,
    isTitleSuggestEnabled,
    titleInputKey,
    isLoading,
    handleSuggest,
    handleChange,
  } = useLibraryPageTitleBlock();

  function handleChangeAndReset(value: string) {
    handleChange(value);
  }

  return (
    <LStack gap="2">
      <LStack minW="0">
        <WStack alignItems="end">
          <HeadingInput
            key={`title:${titleInputKey}`}
            id="name-input"
            size={"2xl" as any}
            fontWeight="bold"
            placeholder="Name..."
            onValueChange={handleChangeAndReset}
            defaultValue={defaultValue}
          />
          {isTitleSuggestEnabled && (
            <IntelligenceAction
              title="Suggest a title for this page"
              onClick={handleSuggest}
              variant="subtle"
              h="full"
              minH="8"
              loading={isLoading}
            />
          )}
        </WStack>
      </LStack>
    </LStack>
  );
}

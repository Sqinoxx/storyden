"use client";

import { useState, useCallback } from "react";
import { Download, Eye } from "lucide-react";

import { IntelligenceAction } from "@/components/site/Action/Intelligence";
import { Heading } from "@/components/ui/heading";
import { HeadingInput } from "@/components/ui/heading-input";
import { HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { getAssetURL, getCleanFilename } from "@/utils/asset";
import { FilePreviewModal, isPreviewableAsset } from "@/components/post/FilePreviewModal";

import { useWatch } from "../../store";
import { useEditState } from "../../useEditState";

import { useLibraryPageTitleBlock } from "./useLibraryPageTitleBlock";

export function LibraryPageTitleBlock() {
  const name = useWatch((s) => s.draft.name);
  const assets = useWatch((s) => s.draft.assets);
  const { editing } = useEditState();

  const [previewOpen, setPreviewOpen] = useState(false);

  if (editing) {
    return <LibraryPageTitleBlockEditing />;
  }

  const fileAsset = assets && assets.length > 0 ? assets[0] : undefined;
  const downloadUrl = fileAsset ? getAssetURL(fileAsset.path) : null;
  const displayName = getCleanFilename(fileAsset?.filename) || fileAsset?.filename || name;
  const canPreview = fileAsset
    ? isPreviewableAsset(fileAsset.mime_type, fileAsset.filename)
    : false;

  async function handleDownload(e: React.MouseEvent) {
    e.preventDefault();
    if (!downloadUrl) return;
    try {
      const res = await fetch(downloadUrl);
      if (!res.ok) throw new Error();
      const blob = await res.blob();
      const blobUrl = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = blobUrl;
      link.download = displayName;
      document.body.appendChild(link);
      link.click();
      link.remove();
      URL.revokeObjectURL(blobUrl);
    } catch {
      const link = document.createElement("a");
      link.href = downloadUrl;
      link.download = displayName;
      document.body.appendChild(link);
      link.click();
      link.remove();
    }
  }

  return (
    <>
      <LStack gap="2">
        <Heading fontSize="heading.2" fontWeight="bold">
          {name || "(untitled)"}
        </Heading>

        {downloadUrl && (
          <HStack gap="2">
            <styled.div
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
              onClick={(e) => {
                e.preventDefault();
                if (canPreview) {
                  setPreviewOpen(true);
                } else {
                  handleDownload(e);
                }
              }}
              title={canPreview ? `${displayName} – Vorschau öffnen` : `${displayName} – Herunterladen`}
              style={{ cursor: "pointer" }}
              _hover={{ bgColor: "bg.muted" }}
            >
              <styled.span flexShrink="0" fontSize="sm">
                📄
              </styled.span>
              <span>{displayName}</span>
              <styled.span display="inline-flex" alignItems="center" gap="1.5" color="fg.muted" flexShrink="0" ml="1">
                {canPreview && (
                  <styled.button
                    type="button"
                    onClick={(e) => { e.preventDefault(); e.stopPropagation(); setPreviewOpen(true); }}
                    title="Vorschau"
                    display="inline-flex"
                    alignItems="center"
                    justifyContent="center"
                    color="fg.muted"
                    _hover={{ color: "fg.default" }}
                    style={{
                      cursor: "pointer",
                      background: "transparent",
                      border: "none",
                      padding: "2px",
                    }}
                  >
                    <Eye size={14} />
                  </styled.button>
                )}
                <styled.button
                  type="button"
                  onClick={handleDownload}
                  title="Herunterladen"
                  display="inline-flex"
                  alignItems="center"
                  justifyContent="center"
                  color="fg.muted"
                  _hover={{ color: "fg.default" }}
                  style={{
                    cursor: "pointer",
                    background: "transparent",
                    border: "none",
                    padding: "2px",
                  }}
                >
                  <Download size={14} />
                </styled.button>
              </styled.span>
            </styled.div>
          </HStack>
        )}
      </LStack>

      {previewOpen && downloadUrl && (
        <FilePreviewModal
          url={downloadUrl}
          displayName={displayName}
          mimeType={fileAsset?.mime_type}
          onClose={() => setPreviewOpen(false)}
          onDownload={() => handleDownload({ preventDefault: () => {} } as React.MouseEvent)}
        />
      )}
    </>
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

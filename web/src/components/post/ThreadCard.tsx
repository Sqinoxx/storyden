import Link from "next/link";
import React, { memo } from "react";
import { FileIcon, Download } from "lucide-react";

import {
  Asset,
  Permission,
  ThreadReference,
  Visibility,
} from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { Byline } from "@/components/content/Byline";
import { CollectionMenu } from "@/components/content/CollectionMenu/CollectionMenu";

import { Card } from "@/components/ui/rich-card";
import { Box, HStack, LStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";
import { timestamp } from "@/utils/date";
import { hasPermission } from "@/utils/permissions";

import { CategoryBadge } from "../category/CategoryBadge";
import { PostReviewBadge } from "../thread/PostReviewBadge";
import { ThreadMenu } from "../thread/ThreadMenu/ThreadMenu";
import {
  DiscussionIcon,
  DiscussionParticipatingIcon,
} from "../ui/icons/Discussion";
import { PinIcon } from "../ui/icons/Pin";

import { LikeButton } from "./LikeButton/LikeButton";
import { useThreadCardModeration } from "./useThreadCardModeration";

/** Returns true if the asset is a document (not an image). */
function isDocumentAsset(asset: Asset) {
  if (!asset) return false;
  if (asset.mime_type && asset.mime_type.startsWith("image/")) {
    return false;
  }
  return true;
}

function isDocumentHref(href: string | null): boolean {
  if (!href) return false;
  const lower = href.toLowerCase();
  const hasDocExt =
    /\.(pdf|docx?|xlsx?|pptx?|zip|rar|7z|txt|csv)(\?.*)?$/i.test(lower);
  const isAssetUrl =
    lower.includes("/api/assets/") || lower.includes("/assets/");
  const isImage = /\.(png|jpe?g|gif|webp|svg|bmp)(\?.*)?$/i.test(lower);
  return (hasDocExt || isAssetUrl) && !isImage;
}

/** Extracts document assets from thread.assets and parses fallback links from thread.body if necessary. */
function extractDocumentAssetsFromThread(thread: ThreadReference): Asset[] {
  const assets: Asset[] = [...(thread.assets ?? []).filter(isDocumentAsset)];
  const existingPaths = new Set(assets.map((a) => a.path));
  const existingNames = new Set(assets.map((a) => a.filename.toLowerCase()));

  if (thread.body && typeof document !== "undefined") {
    try {
      const parsed = new DOMParser().parseFromString(thread.body, "text/html");
      const anchors = parsed.querySelectorAll("a[href]");
      anchors.forEach((a) => {
        const href = a.getAttribute("href");
        if (href && isDocumentHref(href)) {
          const filename =
            a.getAttribute("data-filename") ||
            a.textContent?.trim() ||
            href.split("/").pop() ||
            "Attachment";
          const path = href.replace(/^.*\/api\/assets\//, "");

          if (
            !existingPaths.has(path) &&
            !existingPaths.has(href) &&
            !existingNames.has(filename.toLowerCase())
          ) {
            existingPaths.add(path);
            existingNames.add(filename.toLowerCase());
            assets.push({
              id: path,
              filename,
              path:
                href.startsWith("/") || href.startsWith("http")
                  ? href
                  : `/api/assets/${path}`,
              mime_type: filename.endsWith(".pdf")
                ? "application/pdf"
                : "application/octet-stream",
              size: 0,
              created_at: thread.createdAt,
            } as unknown as Asset);
          }
        }
      });
    } catch {
      // Ignore DOMParser errors
    }
  }

  return assets;
}

/** A compact, styled badge for a file attachment shown in the feed card. */
function FileAttachmentBadge({ asset }: { asset: Asset }) {
  const url = getAssetURL(asset.path);

  function handleDownload(e: React.MouseEvent) {
    e.stopPropagation();
  }

  return (
    <styled.a
      href={url}
      download={asset.filename}
      target="_blank"
      rel="noopener noreferrer"
      display="inline-flex"
      alignItems="center"
      gap="2"
      px="3"
      py="2"
      borderRadius="md"
      borderWidth="thin"
      borderColor="border.default"
      bgColor="bg.subtle"
      color="fg.default"
      textDecoration="none"
      fontSize="sm"
      fontWeight="medium"
      maxWidth="xs"
      overflow="hidden"
      position="relative"
      cursor="pointer"
      _hover={{ bgColor: "bg.muted" }}
      onClick={handleDownload}
      style={{ zIndex: 10, transition: "background 0.15s" }}
    >
      <styled.span
        display="flex"
        alignItems="center"
        justifyContent="center"
        bgColor="bg.muted"
        borderRadius="sm"
        p="1"
        flexShrink="0"
      >
        <FileIcon size={16} />
      </styled.span>
      <styled.span
        overflow="hidden"
        textOverflow="ellipsis"
        whiteSpace="nowrap"
        flex="1"
        minWidth="0"
      >
        {asset.filename}
      </styled.span>
      <styled.span display="flex" alignItems="center" color="fg.muted" flexShrink="0" ml="1">
        <Download size={14} />
      </styled.span>
    </styled.a>
  );
}

type Props = {
  thread: ThreadReference;
  hideCategoryBadge?: boolean;
};

export const ThreadReferenceCard = memo(
  ({ thread, hideCategoryBadge = false }: Props) => {
    const session = useSession();
    const isDraft = thread.visibility === Visibility.draft;
    const permalink = isDraft ? `/new?id=${thread.id}` : `/t/${thread.slug}`;
    const isModerator = hasPermission(
      session,
      Permission.MANAGE_POSTS,
      Permission.ADMINISTRATOR,
    );

    const { isConfirmingDelete, handlers } = useThreadCardModeration(thread);

    const title = thread.title || thread.link?.title || "Untitled post";

    const hasReplied = thread.reply_status.replied > 0;
    const replyCount = thread.reply_status.replies;
    const replyCountLabel =
      replyCount === 1 ? `1 reply` : `${replyCount} replies`;

    const replyStatusLabel = hasReplied
      ? `${replyCountLabel} (including you!)`
      : replyCountLabel;

    const newRepliesCount = thread.read_status?.replies_since ?? 0;
    const lastReadAt = thread.read_status?.last_read_at;
    const newRepliesLabel =
      newRepliesCount > 0 && lastReadAt
        ? `${newRepliesCount} ${newRepliesCount === 1 ? "reply" : "replies"} since you last visited ${timestamp(lastReadAt, false)} ago`
        : undefined;

    const isInReview = thread.visibility === Visibility.review;
    const isPinned = (thread.pinned ?? 0) > 0;
    const cardBackground = isPinned ? "emphasized" : "default";

    // Separate image assets from document assets.
    const documentAssets = extractDocumentAssetsFromThread(thread);
    const imageAssets = (thread.assets ?? []).filter(
      (a) => !isDocumentAsset(a),
    );

    // Use only image assets (or link preview image) as the card cover.
    // Document assets are shown as file badges, not as broken cover images.
    const image = isInReview
      ? undefined
      : getAssetURL(
          imageAssets[0]?.path ?? thread.link?.primary_image?.path,
        );

    // Suppress plain-text description if description is just the document filename
    // to avoid displaying duplicate plain text above the styled file attachment badge.
    let textDescription = thread.description?.trim();
    if (textDescription && documentAssets.length > 0) {
      const isFileNameOnly = documentAssets.some(
        (a) =>
          textDescription === a.filename ||
          textDescription === a.filename.trim() ||
          textDescription?.toLowerCase() === a.filename.toLowerCase() ||
          textDescription === thread.title,
      );
      if (isFileNameOnly) {
        textDescription = undefined;
      }
    }

    return (
      <Card
        shape="responsive"
        backgroundColor={cardBackground}
        id={thread.id}
        title={title}
        titleIcon={isPinned ? <PinIcon w="4" /> : undefined}
        text={textDescription}
        url={permalink}
        image={image}
        controls={
          <HStack>
            {!hideCategoryBadge && thread.category && (
              <CategoryBadge category={thread.category} />
            )}
            {isInReview ? (
              <>
                <PostReviewBadge
                  isModerator={isModerator}
                  postId={thread.id}
                  onAccept={handlers.handleAcceptThread}
                  onEditAndAccept={handlers.handleEditAndAccept}
                  onDelete={handlers.handleConfirmDelete}
                  isConfirmingDelete={isConfirmingDelete}
                  onCancelDelete={handlers.handleCancelDelete}
                />
              </>
            ) : (
              <>
                <LikeButton thread={thread} showCount />
                {session && (
                  <CollectionMenu account={session} thread={thread} />
                )}
              </>
            )}
            <ThreadMenu thread={thread} />
          </HStack>
        }
      >
        <LStack gap="2" w="full">
          {documentAssets.length > 0 && (
            <styled.div
              display="flex"
              flexWrap="wrap"
              gap="2"
              position="relative"
              style={{ zIndex: 1 }}
            >
              {documentAssets.map((asset) => (
                <FileAttachmentBadge
                  key={asset.id || asset.filename}
                  asset={asset}
                />
              ))}
            </styled.div>
          )}
          <Byline
            href={permalink}
            author={thread.author}
            time={new Date(thread.createdAt)}
            updated={new Date(thread.updatedAt)}
            more={
              <Box
                className="thread-byline__more"
                flexShrink="0"
                overflow="hidden"
              >
                <Link
                  className="thread-byline__anchor"
                  href={permalink}
                  title={replyStatusLabel}
                >
                  <styled.span
                    className="thread-byline__reply-status-label"
                    color="fg.muted"
                    display="flex"
                    gap="0.5"
                    alignItems="center"
                  >
                    {hasReplied ? (
                      <DiscussionParticipatingIcon width="4" />
                    ) : (
                      <DiscussionIcon width="4" />
                    )}
                    {replyCount}
                    {newRepliesCount > 0 && (
                      <styled.span
                        color="fg.muted"
                        fontSize="xs"
                        title={newRepliesLabel}
                      >
                        +{newRepliesCount}
                      </styled.span>
                    )}
                  </styled.span>
                </Link>
              </Box>
            }
          />
        </LStack>
      </Card>
    );
  },
);

ThreadReferenceCard.displayName = "ThreadReferenceCard";


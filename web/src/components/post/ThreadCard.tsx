"use client";

import Link from "next/link";
import { memo } from "react";

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
import {
  FileAttachmentBadge,
  isDocumentAsset,
} from "./FileAttachmentList";
import { getAssetURL } from "@/utils/asset";
import { timestamp } from "@/utils/date";
import { hasPermission } from "@/utils/permissions";
import { useLanguage } from "@/lib/i18n";

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

  if (thread.body) {
    try {
      const anchorRegex = /<a\b([^>]*)>(.*?)<\/a>/gi;
      let match: RegExpExecArray | null;
      while ((match = anchorRegex.exec(thread.body)) !== null) {
        const attrString = match[1] ?? "";
        const innerText = (match[2] ?? "").replace(/<[^>]+>/g, "").trim();

        const hrefMatch = /href=["']([^"']+)["']/i.exec(attrString);
        const href = hrefMatch ? hrefMatch[1] : null;

        if (href && isDocumentHref(href)) {
          const filenameMatch =
            /data-filename=["']([^"']+)["']/i.exec(attrString);
          const filenameAttr = filenameMatch ? filenameMatch[1] : null;
          const filename =
            filenameAttr ||
            innerText ||
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
      }
    } catch {
      // Ignore regex errors
    }
  }

  if (assets.length > 1) {
    const hasNamedAsset = assets.some(
      (a) => a.filename && !a.filename.toLowerCase().endsWith("-untitled") && a.filename.toLowerCase() !== "untitled"
    );
    if (hasNamedAsset) {
      return assets.filter(
        (a) => a.filename && !a.filename.toLowerCase().endsWith("-untitled") && a.filename.toLowerCase() !== "untitled"
      );
    }
  }

  return assets;
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
    const { t } = useLanguage();

    const title = thread.title || thread.link?.title || t.thread.untitled;

    const hasReplied = thread.reply_status.replied > 0;
    const replyCount = thread.reply_status.replies;
    const replyCountLabel =
      replyCount === 1 ? `1 ${t.thread.reply}` : `${replyCount} ${t.thread.replies}`;

    const replyStatusLabel = hasReplied
      ? `${replyCountLabel} ${t.thread.includingYou}`
      : replyCountLabel;

    const newRepliesCount = thread.read_status?.replies_since ?? 0;
    const lastReadAt = thread.read_status?.last_read_at;
    const newRepliesLabel =
      newRepliesCount > 0 && lastReadAt
        ? `${newRepliesCount} ${newRepliesCount === 1 ? t.thread.reply : t.thread.replies} ${t.thread.sinceLastVisit} ${timestamp(lastReadAt, false)} ${t.thread.ago}`
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


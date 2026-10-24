"use client";

import { Controller, ControllerProps } from "react-hook-form";
import { match } from "ts-pattern";

import { Asset, Thread, Visibility } from "@/api/openapi-schema";
import { CategoryBadge } from "@/components/category/CategoryBadge";
import { Byline } from "@/components/content/Byline";
import { ContentComposer } from "@/components/content/ContentComposer/ContentComposer";
import { LinkCard } from "@/components/library/links/LinkCard";
import { FileAttachmentList } from "@/components/post/FileAttachmentList";
import { CancelAction } from "@/components/site/Action/Cancel";
import { SaveAction } from "@/components/site/Action/Save";
import { PaginationControls } from "@/components/site/PaginationControls/PaginationControls";
import { Unready } from "@/components/site/Unready";
import { TagBadgeList } from "@/components/tag/TagBadgeList";
import { Breadcrumbs } from "@/components/thread/Breadcrumbs";
import { PostReviewBadge } from "@/components/thread/PostReviewBadge";
import { ReplyBox } from "@/components/thread/ReplyBox/ReplyBox";
import { ReplyProvider } from "@/components/thread/ReplyContext";
import { ReplyList } from "@/components/thread/ReplyList/ReplyList";
import { SemesterSelect } from "@/components/thread/SemesterSelect/SemesterSelect";
import { Signature } from "@/components/thread/Signature";
import { ThreadDeletedAlert } from "@/components/thread/ThreadDeletedAlert";
import { ThreadMenu } from "@/components/thread/ThreadMenu/ThreadMenu";
import { TagListField } from "@/components/thread/ThreadTagList";
import { FormErrorText } from "@/components/ui/FormErrorText";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { HeadingInput } from "@/components/ui/heading-input";
import { CheckIcon } from "@/components/ui/icons/Check";
import {
  DiscussionIcon,
  DiscussionParticipatingIcon,
} from "@/components/ui/icons/Discussion";
import { LikeIcon, LikeSavedIcon } from "@/components/ui/icons/Like";
import {
  SortAscendingIcon,
  SortDescendingIcon,
} from "@/components/ui/icons/Sort";
import * as Menu from "@/components/ui/menu";
import { VisibilityBadge } from "@/components/visibility/VisibilityBadge";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, VStack, WStack, styled } from "@/styled-system/jsx";
import { CardBox } from "@/styled-system/patterns";

import { Form, Props, useThreadScreen } from "./useThreadScreen";

export function ThreadScreen(props: Props) {
  const t = useTranslation();
  const {
    ready,
    error,
    form,
    isEditing,
    isEmpty,
    isUploading,
    resetKey,
    attachments,
    isModerator,
    isConfirmingDelete,
    sortOrder,
    canLike,
    data,
    handlers,
  } = useThreadScreen(props);

  if (!ready) {
    return <Unready error={error} />;
  }

  const { thread } = data;
  const signatureConfig = props.initialSignatureConfig ?? {
    enabled: true,
    maxHeight: 160,
  };

  return (
    <ReplyProvider>
      <LStack gap="6" pb="12">
        <HStack w="full" gap="3">
          <Breadcrumbs thread={thread} />

          <HStack gap="1" flexShrink="0">
            {isEditing && (
              <>
                <CancelAction
                  type="button"
                  onClick={handlers.handleDiscardChanges}
                >
                  {t.common.discard}
                </CancelAction>
                <SaveAction type="submit" disabled={isEmpty || isUploading}>
                  {t.common.save}
                </SaveAction>
              </>
            )}

            <ThreadMenu thread={thread} editingEnabled movingEnabled />
          </HStack>
        </HStack>

        <styled.article className={CardBox()} p="4" gap="4">
          <styled.form
            display="flex"
            flexDirection="column"
            alignItems="start"
            gap="4"
            width="full"
            onSubmit={handlers.handleSave}
          >
            {thread.deletedAt !== undefined && (
              <ThreadDeletedAlert thread={thread} />
            )}

            <LStack gap="3">
              <WStack alignItems="center" flexWrap="wrap" rowGap="2">
                <HStack gap="2" flexWrap="wrap" minW="0">
                  <Byline
                    href={`#${thread.id}`}
                    author={thread.author}
                    time={new Date(thread.createdAt)}
                    updated={new Date(thread.updatedAt)}
                  />

                  {thread.category && (
                    <CategoryBadge category={thread.category} />
                  )}

                  {!isEditing && <TagBadgeList tags={thread.tags} />}
                </HStack>

                {match(thread.visibility)
                  .with(Visibility.published, () => null)
                  .with(Visibility.review, () => (
                    <PostReviewBadge
                      isModerator={isModerator}
                      postId={thread.id}
                      onAccept={handlers.handleAcceptThread}
                      onEditAndAccept={handlers.handleEditAndAccept}
                      onDelete={handlers.handleConfirmDelete}
                      isConfirmingDelete={isConfirmingDelete}
                      onCancelDelete={handlers.handleCancelDelete}
                    />
                  ))
                  .otherwise(() => (
                    <VisibilityBadge visibility={thread.visibility} />
                  ))}
              </WStack>
              <FormErrorText>
                {form.formState.errors.root?.message}
              </FormErrorText>

              {isEditing ? (
                <TitleInput name="title" control={form.control} />
              ) : (
                <Heading
                  fontSize="heading.variable.1"
                  fontWeight="bold"
                  lineHeight="tight"
                >
                  {thread.title}
                </Heading>
              )}

              {isEditing && (
                <HStack gap="2" flexWrap="wrap">
                  <SemesterSelect control={form.control} name="semester" />
                  <TagListField
                    name="tags"
                    control={form.control}
                    initialTags={thread.tags}
                    attachments={thread.assets}
                  />
                </HStack>
              )}
            </LStack>

            {thread.link && <LinkCard link={thread.link} />}

            <ThreadBodyInput
              control={form.control}
              name="body"
              initialValue={thread.body}
              resetKey={resetKey}
              disabled={!isEditing}
              handleEmptyStateChange={handlers.handleEmptyStateChange}
              handleAssetUpload={handlers.handleAssetUpload}
              handleUploadingChange={handlers.handleUploadingChange}
            />

            {((isEditing && attachments.length > 0) ||
              (thread.assets && thread.assets.length > 0)) && (
              <FileAttachmentList
                assets={
                  isEditing
                    ? attachments
                    : (data.thread.assets ?? thread.assets)
                }
                body={isEditing ? form.watch("body") : thread.body}
                onRemoveAsset={
                  isEditing ? handlers.handleRemoveAsset : undefined
                }
              />
            )}

            {signatureConfig.enabled && (
              <Signature
                signature={thread.author.signature}
                maxHeight={signatureConfig.maxHeight}
              />
            )}
          </styled.form>

          <ThreadStats
            thread={thread}
            sortOrder={sortOrder}
            onSortOrderChange={handlers.handleSetSortOrder}
            canLike={canLike}
            onToggleLike={handlers.handleToggleLike}
          />
        </styled.article>

        <ReplyBox
          initialSession={props.initialSession}
          initialSettings={props.initialSettings}
          thread={thread}
          sortOrder={sortOrder}
        />

        <VStack w="full">
          {data.thread.replies.total_pages > 1 && (
            <PaginationControls
              path={`/t/${thread.slug}`}
              params={sortOrder === "asc" ? { sort: "asc" } : undefined}
              currentPage={data.thread.replies.current_page ?? 1}
              totalPages={data.thread.replies.total_pages}
              pageSize={data.thread.replies.page_size}
            />
          )}

          <ReplyList
            initialSession={props.initialSession}
            thread={thread}
            currentPage={data.thread.replies.current_page}
            initialSignatureConfig={signatureConfig}
          />

          {data.thread.replies.total_pages > 1 && (
            <PaginationControls
              path={`/t/${thread.slug}`}
              params={sortOrder === "asc" ? { sort: "asc" } : undefined}
              currentPage={data.thread.replies.current_page ?? 1}
              totalPages={data.thread.replies.total_pages}
              pageSize={data.thread.replies.page_size}
            />
          )}
        </VStack>
      </LStack>
    </ReplyProvider>
  );
}

type TitleInputProps = Omit<ControllerProps<Form>, "render">;

export function TitleInput({ control }: TitleInputProps) {
  const t = useTranslation();
  return (
    <Controller<Form>
      render={({ field: { onChange, ...field }, formState, fieldState }) => {
        return (
          <>
            <HeadingInput
              id="title-input"
              placeholder={t.editor.titlePlaceholder}
              onValueChange={onChange}
              defaultValue={formState.defaultValues?.["title"]}
              {...field}
            />

            <FormErrorText>{fieldState.error?.message}</FormErrorText>
          </>
        );
      }}
      control={control}
      name="title"
    />
  );
}

type ThreadBodyInputProps = Omit<ControllerProps<Form, "body">, "render"> & {
  initialValue: string;
  resetKey: string;
  handleEmptyStateChange: (isEmpty: boolean) => void;
  handleAssetUpload: (asset: Asset) => void;
  handleUploadingChange: (uploading: boolean) => void;
};

function ThreadBodyInput({
  control,
  name = "body",
  initialValue,
  resetKey,
  disabled,
  handleEmptyStateChange,
  handleAssetUpload,
  handleUploadingChange,
}: ThreadBodyInputProps) {
  return (
    <Controller<Form, "body">
      render={({ field: { onChange, value } }) => {
        function handleChange(val: string, isEmpty: boolean) {
          handleEmptyStateChange(isEmpty);
          onChange(val);
        }

        return (
          <ContentComposer
            initialValue={initialValue}
            value={typeof value === "string" ? value : undefined}
            onChange={handleChange}
            resetKey={resetKey}
            disabled={disabled}
            onAssetUpload={handleAssetUpload}
            onUploadingChange={handleUploadingChange}
          />
        );
      }}
      control={control}
      name={name}
    />
  );
}

function ThreadStats({
  thread,
  sortOrder,
  onSortOrderChange,
  canLike,
  onToggleLike,
}: {
  thread: Thread;
  sortOrder: "asc" | "desc";
  onSortOrderChange: (order: "asc" | "desc") => void;
  canLike: boolean;
  onToggleLike: () => void;
}) {
  const t = useTranslation();
  const likeCount = thread.likes.likes;
  const likeLabel = likeCount === 1 ? t.thread.like : t.thread.likes;
  const replyCount = thread.reply_status.replies;
  const replyLabel = replyCount === 1 ? t.thread.reply : t.thread.replies;

  return (
    <WStack
      justifyContent="space-between"
      alignItems="center"
      width="full"
      pt="3"
      borderTopWidth="thin"
      borderTopColor="border.subtle"
      color="fg.muted"
      fontSize="sm"
    >
      <HStack gap="4">
        <Button
          type="button"
          variant="ghost"
          size="xs"
          ms="-2"
          gap="1"
          fontSize="sm"
          fontWeight="normal"
          color={thread.likes.liked ? "fg.default" : "fg.muted"}
          aria-pressed={thread.likes.liked}
          title={
            thread.likes.liked ? t.thread.unlikeAction : t.thread.likeAction
          }
          disabled={!canLike}
          onClick={onToggleLike}
        >
          {thread.likes.liked ? (
            <LikeSavedIcon width="4" />
          ) : (
            <LikeIcon width="4" />
          )}
          <span>
            {likeCount} {likeLabel}
          </span>
        </Button>

        <styled.span
          display="flex"
          gap="1"
          alignItems="center"
          title={
            thread.reply_status.replied ? t.thread.youRepliedToThis : undefined
          }
        >
          <span>
            {thread.reply_status.replied ? (
              <DiscussionParticipatingIcon width="4" />
            ) : (
              <DiscussionIcon width="4" />
            )}
          </span>
          <span>
            {replyCount} {replyLabel}
          </span>
        </styled.span>
      </HStack>

      <Menu.Root positioning={{ placement: "bottom-end" }} lazyMount>
        <Menu.Trigger asChild>
          <Button variant="ghost" size="xs" aria-label={t.thread.sortBy}>
            <HStack gap="1" fontSize="xs">
              {sortOrder === "asc" ? (
                <SortAscendingIcon width="3.5" height="3.5" />
              ) : (
                <SortDescendingIcon width="3.5" height="3.5" />
              )}
              <span>
                {sortOrder === "asc"
                  ? t.thread.sortOldestFirst
                  : t.thread.sortNewestFirst}
              </span>
            </HStack>
          </Button>
        </Menu.Trigger>
        <Menu.Positioner>
          <Menu.Content minW="44">
            <Menu.ItemGroup id="reply-sort-options">
              <Menu.Item
                key="asc"
                value="asc"
                onClick={() => onSortOrderChange("asc")}
                aria-label={t.thread.sortOldestFirst}
              >
                <HStack gap="2" justifyContent="space-between" w="full">
                  <HStack gap="2">
                    <SortAscendingIcon width="4" height="4" />
                    <span>{t.thread.sortOldestFirst}</span>
                  </HStack>
                  {sortOrder === "asc" && <CheckIcon width="4" height="4" />}
                </HStack>
              </Menu.Item>
              <Menu.Item
                key="desc"
                value="desc"
                onClick={() => onSortOrderChange("desc")}
                aria-label={t.thread.sortNewestFirst}
              >
                <HStack gap="2" justifyContent="space-between" w="full">
                  <HStack gap="2">
                    <SortDescendingIcon width="4" height="4" />
                    <span>{t.thread.sortNewestFirst}</span>
                  </HStack>
                  {sortOrder === "desc" && <CheckIcon width="4" height="4" />}
                </HStack>
              </Menu.Item>
            </Menu.ItemGroup>
          </Menu.Content>
        </Menu.Positioner>
      </Menu.Root>
    </WStack>
  );
}

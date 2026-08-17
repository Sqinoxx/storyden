"use client";

import { Controller, ControllerProps } from "react-hook-form";
import { match } from "ts-pattern";

import { Unready } from "@/components/site/Unready";

import { Thread, Visibility } from "@/api/openapi-schema";
import { CategoryBadge } from "@/components/category/CategoryBadge";
import { Byline } from "@/components/content/Byline";
import { ContentComposer } from "@/components/content/ContentComposer/ContentComposer";
import { LinkCard } from "@/components/library/links/LinkCard";
import { CancelAction } from "@/components/site/Action/Cancel";
import { SaveAction } from "@/components/site/Action/Save";
import { PaginationControls } from "@/components/site/PaginationControls/PaginationControls";
import { TagBadgeList } from "@/components/tag/TagBadgeList";
import { Breadcrumbs } from "@/components/thread/Breadcrumbs";
import { PostReviewBadge } from "@/components/thread/PostReviewBadge";
import { ReplyBox } from "@/components/thread/ReplyBox/ReplyBox";
import { ReplyProvider } from "@/components/thread/ReplyContext";
import { ReplyList } from "@/components/thread/ReplyList/ReplyList";
import { Signature } from "@/components/thread/Signature";
import { ThreadDeletedAlert } from "@/components/thread/ThreadDeletedAlert";
import { ThreadMenu } from "@/components/thread/ThreadMenu/ThreadMenu";
import { TagListField } from "@/components/thread/ThreadTagList";
import { FormErrorText } from "@/components/ui/FormErrorText";
import { Heading } from "@/components/ui/heading";
import { useTranslation } from "@/lib/i18n";
import { Button } from "@/components/ui/button";
import { CheckIcon } from "@/components/ui/icons/Check";
import { SortAscendingIcon, SortDescendingIcon } from "@/components/ui/icons/Sort";
import * as Menu from "@/components/ui/menu";
import { HeadingInput } from "@/components/ui/heading-input";
import {
  DiscussionIcon,
  DiscussionParticipatingIcon,
} from "@/components/ui/icons/Discussion";
import { LikeIcon, LikeSavedIcon } from "@/components/ui/icons/Like";
import { VisibilityBadge } from "@/components/visibility/VisibilityBadge";
import { HStack, LStack, VStack, WStack, styled } from "@/styled-system/jsx";
import { FileAttachmentList } from "@/components/post/FileAttachmentList";

import { Form, Props, useThreadScreen } from "./useThreadScreen";

export function ThreadScreen(props: Props) {
  const {
    ready,
    error,
    form,
    isEditing,
    isEmpty,
    resetKey,
    attachments,
    isModerator,
    isConfirmingDelete,
    sortOrder,
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
      <LStack gap="4" pb="12">
        <styled.form
          display="flex"
          flexDirection="column"
          alignItems="start"
          gap="1"
          width="full"
          onSubmit={handlers.handleSave}
        >
          <WStack alignItems="start">
            <Breadcrumbs thread={thread} />

            <HStack>
              {isEditing && (
                <>
                  <CancelAction
                    type="button"
                    onClick={handlers.handleDiscardChanges}
                  >
                    Discard
                  </CancelAction>
                  <SaveAction type="submit" disabled={isEmpty}>
                    Save
                  </SaveAction>
                </>
              )}

              <ThreadMenu thread={thread} editingEnabled movingEnabled />
            </HStack>
          </WStack>

          {thread.deletedAt !== undefined && (
            <ThreadDeletedAlert thread={thread} />
          )}
          <WStack justifyContent="space-between">
            <HStack>
              <Byline
                href={`#${thread.id}`}
                author={thread.author}
                time={new Date(thread.createdAt)}
                updated={new Date(thread.updatedAt)}
              />

              {thread.category && <CategoryBadge category={thread.category} />}
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
          <FormErrorText>{form.formState.errors.root?.message}</FormErrorText>

          {isEditing ? (
            <TitleInput name="title" control={form.control} />
          ) : (
            <Heading fontSize="heading.variable.1" fontWeight="bold">
              {thread.title}
            </Heading>
          )}

          {isEditing ? (
            <TagListField
              name="tags"
              control={form.control}
              initialTags={thread.tags}
              attachments={thread.assets}
            />
          ) : (
            <TagBadgeList tags={thread.tags} />
          )}

          {thread.link && <LinkCard link={thread.link} />}

          <ThreadBodyInput
            control={form.control}
            name="body"
            initialValue={thread.body}
            resetKey={resetKey}
            disabled={!isEditing}
            handleEmptyStateChange={handlers.handleEmptyStateChange}
          />

          {((isEditing && attachments.length > 0) || (thread.assets && thread.assets.length > 0)) && (
            <FileAttachmentList
              assets={isEditing ? attachments : (data.thread.assets ?? thread.assets)}
              body={isEditing ? form.watch("body") : thread.body}
              onRemoveAsset={isEditing ? handlers.handleRemoveAsset : undefined}
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
        />

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
  return (
    <Controller<Form>
      render={({ field: { onChange, ...field }, formState, fieldState }) => {
        return (
          <>
            <HeadingInput
              id="title-input"
              placeholder="Thread title..."
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
};

function ThreadBodyInput({
  control,
  name = "body",
  initialValue,
  resetKey,
  disabled,
  handleEmptyStateChange,
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
}: {
  thread: Thread;
  sortOrder: "asc" | "desc";
  onSortOrderChange: (order: "asc" | "desc") => void;
}) {
  const t = useTranslation();
  const likeCount = thread.likes.likes;
  const likeLabel = likeCount === 1 ? t.thread.like : t.thread.likes;
  const replyCount = thread.reply_status.replies;
  const replyLabel = replyCount === 1 ? t.thread.reply : t.thread.replies;

  return (
    <WStack justifyContent="space-between" width="full" color="fg.muted">
      <HStack gap="4">
        <styled.span
          display="flex"
          gap="1"
          alignItems="center"
          title={thread.likes.liked ? t.thread.youLikedThis : undefined}
        >
          <span>
            {thread.likes.liked ? (
              <LikeSavedIcon width="4" />
            ) : (
              <LikeIcon width="4" />
            )}
          </span>
          <span>
            {likeCount} {likeLabel}
          </span>
        </styled.span>

        <styled.span
          display="flex"
          gap="1"
          alignItems="center"
          title={
            thread.reply_status.replied
              ? t.thread.youRepliedToThis
              : undefined
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

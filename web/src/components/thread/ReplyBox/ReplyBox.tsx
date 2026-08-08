"use client";

import { useId } from "react";
import Link from "next/link";
import { Controller, ControllerProps } from "react-hook-form";

import { Anchor } from "@/components/site/Anchor";
import { useAttachmentUpload } from "@/components/content/useAttachmentUpload";
import { ContentComposer } from "@/components/content/ContentComposer/ContentComposer";
import { MemberIdent } from "@/components/member/MemberBadge/MemberIdent";
import { Admonition } from "@/components/ui/admonition";
import { Button } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { CloseIcon } from "@/components/ui/icons/Close";
import { DiscussionIcon } from "@/components/ui/icons/Discussion";
import { useTranslation } from "@/lib/i18n";
import { usePublicRegistration } from "@/lib/settings/registration";
import { css } from "@/styled-system/css";
import { Box, HStack, LStack, VStack, WStack, styled } from "@/styled-system/jsx";
import { CardBox } from "@/styled-system/patterns";
import { button } from "@/styled-system/recipes";
import { getAssetURL } from "@/utils/asset";
import { timestamp } from "@/utils/date";

import { useReplyContext } from "../ReplyContext";

import { Form, Props, useReplyBox } from "./useReplyBox";

export function ReplyBox(props: Props) {
  const fileInputId = useId();
  const { replyTo, clearReplyTo } = useReplyContext();
  const t = useTranslation();
  const { isUploading, upload } = useAttachmentUpload();
  const {
    isLoggedIn,
    isEmpty,
    isLoading,
    form,
    resetKey,
    postedReply,
    handlers,
  } = useReplyBox(props);

  if (!isLoggedIn) {
    return <LoginToReply initialSettings={props.initialSettings} />;
  }

  return (
    <VStack w="full" gap="2" alignItems="stretch">
      <Admonition
        value={!!postedReply}
        onChange={handlers.handleReplyPostedAdmonitionClose}
      >
        {postedReply && (
          <LStack h="full" justifyContent="center">
            <styled.p fontSize="sm" color="fg.muted">
              Your reply has been posted on{" "}
              <Link
                className={css({
                  color: "fg.emphasized",
                  _hover: { textDecoration: "underline" },
                })}
                href={postedReply.permalink}
                onClick={handlers.handleReplyNavigation}
              >
                page {postedReply.pageNumber}
              </Link>
              .
            </styled.p>
          </LStack>
        )}
      </Admonition>

      <styled.form
        className={CardBox()}
        display="flex"
        flexDirection="column"
        gap="2"
        width="full"
        onSubmit={handlers.handleSubmit}
      >
        {replyTo && (
          <WStack py="1.5" px="3" borderRadius="md" bgColor="bg.muted" justifyContent="space-between">
            <HStack gap="1" fontSize="sm" color="fg.muted">
              <styled.span>Replying&nbsp;to</styled.span>
              <MemberIdent
                profile={replyTo.reply.author}
                name="handle"
                size="xs"
              />
              <styled.a href={`#${replyTo.reply.id}`}>
                {timestamp(replyTo.reply.createdAt)}
              </styled.a>
            </HStack>

            <IconButton
              type="button"
              size="xs"
              variant="ghost"
              aria-label="Clear reply-to"
              onClick={clearReplyTo}
            >
              <CloseIcon />
            </IconButton>
          </WStack>
        )}

        <HStack gap="1" fontSize="sm" color="fg.muted" fontWeight="medium">
          <styled.span textWrap="nowrap">{t.thread.replyTo}</styled.span>
          <MemberIdent
            profile={props.thread.author}
            name="handle"
            avatar="hidden"
          />
        </HStack>

        <Box w="full">
          <ReplyBodyInput
            name="body"
            control={form.control}
            handleEmptyStateChange={handlers.handleEmptyStateChange}
            resetKey={resetKey}
          />
        </Box>

        <WStack
          w="full"
          pt="2"
          justifyContent="flex-end"
          alignItems="center"
        >
          <HStack gap="2">
            <styled.label
              className={button({ size: "sm", variant: "ghost" })}
              htmlFor={fileInputId}
              title={t.editor.uploadFile}
              aria-disabled={isUploading}
              opacity={isUploading ? "5" : "full"}
              pointerEvents={isUploading ? "none" : "auto"}
            >
              📎 {t.editor.uploadFile}
            </styled.label>
            <styled.input
              id={fileInputId}
              type="file"
              multiple
              display="none"
              accept="*"
              disabled={isUploading}
              onChange={async (e) => {
                const files = Array.from(e.currentTarget.files ?? []);
                e.currentTarget.value = "";

                const uploaded = await upload(files);

                for (const { markup } of uploaded) {
                  const current = form.getValues("body") ?? "";
                  const newBody =
                    current && current !== "<p></p>"
                      ? current + "<p>" + markup + "</p>"
                      : "<p>" + markup + "</p>";

                  form.setValue("body", newBody, {
                    shouldValidate: true,
                    shouldDirty: true,
                  });
                }
              }}
            />

            <Button
              type="submit"
              size="sm"
              variant="subtle"
              disabled={isLoading || isEmpty}
              loading={isLoading}
            >
              {t.thread.replyPost}
            </Button>
          </HStack>
        </WStack>
      </styled.form>
    </VStack>
  );
}

type ReplyBodyInputProps = Omit<ControllerProps<Form>, "render"> & {
  handleEmptyStateChange: (isEmpty: boolean) => void;
  resetKey: string;
};

function ReplyBodyInput({
  control,
  name,
  handleEmptyStateChange,
  resetKey,
}: ReplyBodyInputProps) {
  return (
    <Controller<Form>
      render={({ field: { onChange, value } }) => {
        function handleChange(val: string, isEmpty: boolean) {
          handleEmptyStateChange(isEmpty);
          onChange(val);
        }

        return (
          <ContentComposer
            onChange={handleChange}
            resetKey={resetKey}
            value={value}
          />
        );
      }}
      control={control}
      name={name}
    />
  );
}

function LoginToReply({
  initialSettings,
}: {
  initialSettings?: Props["initialSettings"];
}) {
  const canRegister = usePublicRegistration(initialSettings);
  const action = canRegister ? "sign up or log in" : "log in";

  return (
    <HStack
      w="full"
      p="8"
      borderRadius="xl"
      bgColor="border.muted"
      justifyContent="center"
    >
      <DiscussionIcon width="4" />

      <p>
        Please <Anchor href={canRegister ? "/register" : "/login"}>{action}</Anchor>{" "}
        to reply
      </p>
    </HStack>
  );
}

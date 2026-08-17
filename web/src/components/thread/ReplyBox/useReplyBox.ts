"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { FocusEvent, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { Account, DatagraphItemKind, Thread } from "@/api/openapi-schema";

import { handle } from "@/api/client";
import { useSession } from "@/auth";
import { sendBeacon } from "@/lib/beacon/beacon";
import type { Settings } from "@/lib/settings/settings";
import { useThreadMutations } from "@/lib/thread/mutation";

import { useReplyContext } from "../ReplyContext";

export type Props = {
  initialSession?: Account;
  initialSettings?: Settings;
  thread: Thread;
  sortOrder: "asc" | "desc";
};

type ReplyLocationState = {
  id: string;
  pageNumber: number;
  permalink: string;
};

export const FormSchema = z.object({
  body: z.string().min(1, "Please enter a message."),
});
export type Form = z.infer<typeof FormSchema>;

export function useReplyBox({
  initialSession,
  initialSettings,
  thread,
  sortOrder,
}: Props) {
  const session = useSession(initialSession, initialSettings);
  const { replyTo, clearReplyTo } = useReplyContext();
  const { createReply, revalidate } = useThreadMutations(
    thread,
    thread.replies.current_page,
    thread.replies.total_pages,
    sortOrder,
  );
  const [resetKey, setResetKey] = useState("");
  const [isEmpty, setEmpty] = useState(true);
  const [isFocused, setFocused] = useState(false);
  const [postedReply, setPostedReply] = useState<ReplyLocationState | null>(
    null,
  );
  const form = useForm<Form>({ resolver: zodResolver(FormSchema) });

  function handleEmptyStateChange(isEmpty: boolean) {
    setEmpty(isEmpty);
  }

  function handleFocus() {
    setFocused(true);
  }

  function handleBlur(e: FocusEvent<HTMLFormElement>) {
    if (!e.currentTarget.contains(e.relatedTarget)) {
      setFocused(false);
    }
  }

  function handleReplyPostedAdmonitionClose() {
    setPostedReply(null);
  }

  function handleReplyNavigation() {
    setPostedReply(null);
  }

  const handleSubmit = form.handleSubmit(async (data: Form) => {
    await handle(
      async () => {
        const { id } = await createReply({
          body: data.body,
          reply_to: replyTo?.reply.id,
        });

        // Mark the thread as read after successfully replying to it
        try {
          sendBeacon(DatagraphItemKind.thread, thread.id);
        } catch (error) {
          console.warn("failed to send beacon:", error);
        }

        // This is a little hack tbh, essentially if this prop for the
        // ContentComposer component changes, its value is reset. Could have
        // done it with a hook but... meh this is simpler (albeit not idiomatic)
        setResetKey(new Date().toISOString());
        form.reset();
        setEmpty(true);
        clearReplyTo();

        // If we are not on the page where the new reply appears, we need to
        // inform the user and provide them a link to navigate there. A new
        // reply is the newest one, so it lands on the first page when sorted
        // newest-first, or the last page when sorted oldest-first.
        const currentPage = thread.replies.current_page;
        const totalPages = thread.replies.total_pages;
        const targetPage = sortOrder === "desc" ? 1 : (totalPages ?? 1);
        const isOnTargetPage = !currentPage || currentPage === targetPage;
        if (!isOnTargetPage && totalPages) {
          const params = new URLSearchParams();
          if (targetPage > 1) params.set("page", targetPage.toString());
          if (sortOrder === "asc") params.set("sort", "asc");
          const query = params.toString();

          setPostedReply({
            id,
            pageNumber: targetPage,
            permalink: `/t/${thread.slug}${query ? `?${query}` : ""}#${id}`,
          });
        }
      },
      {
        cleanup: async () => await revalidate(),
      },
    );
  });

  return {
    isLoggedIn: !!session,
    isEmpty,
    isExpanded: isFocused || !isEmpty,
    isLoading: form.formState.isSubmitting,
    resetKey,
    postedReply,
    form,
    handlers: {
      handleSubmit,
      handleEmptyStateChange,
      handleReplyPostedAdmonitionClose,
      handleReplyNavigation,
      handleFocus,
      handleBlur,
    },
  };
}

import { Account, Thread } from "@/api/openapi-schema";

import type { SignatureConfig } from "@/lib/settings/settings";
import { styled } from "@/styled-system/jsx";

import { Reply } from "../Reply/Reply";

type Props = {
  initialSession?: Account;
  thread: Thread;
  currentPage?: number;
  initialSignatureConfig: SignatureConfig;
  sortOrder?: "asc" | "desc";
};

export function ReplyList({
  initialSession,
  thread,
  currentPage,
  initialSignatureConfig,
  sortOrder = "asc",
}: Props) {
  const replies = [...thread.replies.replies];
  if (sortOrder === "desc") {
    replies.reverse();
  }

  return (
    <styled.ol
      listStyleType="none"
      m="0"
      gap="4"
      display="flex"
      flexDir="column"
      width="full"
    >
      {replies.map((reply) => (
        <styled.li key={reply.id} listStyleType="none" m="0">
          <Reply
            initialSession={initialSession}
            thread={thread}
            reply={reply}
            currentPage={currentPage}
            initialSignatureConfig={initialSignatureConfig}
          />
        </styled.li>
      ))}
    </styled.ol>
  );
}

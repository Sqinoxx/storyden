import Link from "next/link";

import { DatagraphItem } from "@/api/openapi-schema";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";

type Props = {
  title: string;
  hint?: string;
  items: DatagraphItem[];
  testId: string;
};

export function SimilarThreadList({ title, hint, items, testId }: Props) {
  const t = useTranslation();
  const threads = items.flatMap((i) => (i.kind === "thread" ? [i.ref] : []));
  if (threads.length === 0) return null;

  return (
    <LStack
      w="full"
      gap="2"
      p="3"
      borderRadius="md"
      borderWidth="thin"
      borderColor="border.subtle"
      bg="bg.muted"
      data-testid={testId}
    >
      <LStack gap="0">
        <styled.p fontSize="sm" fontWeight="medium">
          {title}
        </styled.p>
        {hint && (
          <styled.p fontSize="xs" color="fg.muted">
            {hint}
          </styled.p>
        )}
      </LStack>
      <styled.ul w="full" display="flex" flexDirection="column" gap="1">
        {threads.map((th) => (
          <styled.li key={th.id}>
            <HStack justify="space-between" gap="2" fontSize="sm">
              <Link href={`/t/${th.slug}`}>
                <styled.span
                  color="fg.default"
                  _hover={{ textDecoration: "underline" }}
                >
                  {th.title}
                </styled.span>
              </Link>
              <styled.span color="fg.subtle" fontSize="xs" flexShrink="0">
                {t.thread.similarReplies.replace(
                  "{count}",
                  String(th.reply_status.replies),
                )}
              </styled.span>
            </HStack>
          </styled.li>
        ))}
      </styled.ul>
    </LStack>
  );
}

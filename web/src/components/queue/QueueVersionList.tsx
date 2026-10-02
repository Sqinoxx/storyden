import { NodeDraft } from "@/api/openapi-schema";
import { LibraryPageBadge, LibraryBadge } from "@/components/library/LibraryBadge";
import { MemberBadge } from "@/components/member/MemberBadge/MemberBadge";
import { Timestamp } from "@/components/site/Timestamp";
import { DraftIcon } from "@/components/ui/icons/Draft";
import { LinkButton } from "@/components/ui/link-button";
import { Card, CardRows } from "@/components/ui/rich-card";
import { HStack, WStack } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";
import { useTranslation } from "@/lib/i18n";

export function QueueVersionList({ drafts }: { drafts: NodeDraft[] }) {
  const t = useTranslation();
  if (drafts.length === 0) {
    return <p>{t.queue.editsEmpty}</p>;
  }

  return (
    <CardRows>
      {drafts.map((draft) => (
        <QueueVersionListItem key={draft.id} draft={draft} />
      ))}
    </CardRows>
  );
}

function QueueVersionListItem({ draft }: { draft: NodeDraft }) {
  const t = useTranslation();
  const node = draft.node;
  const url = node.parent
    ? `/l/${node.parent.slug}/${node.slug}`
    : `/l/${node.slug}`;

  return (
    <Card
      key={draft.id}
      id={draft.id}
      shape="responsive"
      url={url}
      image={getAssetURL(node.primary_image?.path)}
      title={node.name}
      text="Draft edit proposal"
      titleIcon={<DraftIcon width="4" height="4" />}
      controls={
        <WStack>
          <HStack gap="2">
            <MemberBadge profile={draft.author} size="sm" />
            <Timestamp created={draft.updated_at} large />

            {node.parent ? (
              <LibraryPageBadge {...node.parent} />
            ) : (
              <LibraryBadge />
            )}
          </HStack>

          <LinkButton
            href={`${url}?version=${draft.id}`}
            size="xs"
            variant="subtle"
          >
            {t.library.review}
          </LinkButton>
        </WStack>
      }
    />
  );
}

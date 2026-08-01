import useSWR from "swr";

import { tagGet } from "@/api/openapi-client/tags";
import {
  DatagraphItem,
  DatagraphItemKind,
  TagReference,
} from "@/api/openapi-schema";
import { DatagraphItemCard } from "@/components/datagraph/DatagraphItemCard";
import { IntelligenceAction } from "@/components/site/Action/Intelligence";
import { TagBadgeList } from "@/components/tag/TagBadgeList";
import { MultiSelectPicker } from "@/components/ui/MultiSelectPicker";
import { HStack, VStack } from "@/styled-system/jsx";

import { useLibraryPageContext } from "../../Context";
import { useWatch } from "../../store";
import { useEditState } from "../../useEditState";

import { useLibraryPageTagsBlockEditing } from "./useLibraryPageTagsBlock";

function TaggedItemsForTags({
  tags,
  nodeID,
}: {
  tags: TagReference[];
  nodeID?: string;
}) {
  const tagNames = tags.map((t) => t.name).filter(Boolean);

  const { data: items = [] } = useSWR(
    tagNames.length > 0 ? ["library-tagged-items", ...tagNames] : null,
    async () => {
      const results = await Promise.all(
        tagNames.map((name) => tagGet(name).catch(() => null)),
      );
      const map = new Map<string, DatagraphItem>();
      for (const res of results) {
        if (res && res.items) {
          for (const item of res.items) {
            if (
              item.kind === DatagraphItemKind.node &&
              item.ref.id === nodeID
            ) {
              continue;
            }
            const key = `${item.kind}:${item.ref.id}`;
            if (!map.has(key)) {
              map.set(key, item);
            }
          }
        }
      }
      return Array.from(map.values());
    },
  );

  if (items.length === 0) {
    return null;
  }

  return (
    <VStack alignItems="start" w="full" gap="2" mt="2">
      {items.map((item) => (
        <DatagraphItemCard key={`${item.kind}-${item.ref.id}`} item={item} />
      ))}
    </VStack>
  );
}

export function LibraryPageTagsBlock() {
  const { isDirectEditing } = useEditState();
  const { nodeID } = useLibraryPageContext();
  const tags = useWatch((s) => s.draft.tags);

  if (isDirectEditing) {
    return <LibraryPageTagsBlockEditing />;
  }

  return (
    <VStack w="full" gap="2" alignItems="start">
      <TagBadgeList tags={tags} />
      <TaggedItemsForTags tags={tags} nodeID={nodeID} />
    </VStack>
  );
}

export function LibraryPageTagsBlockEditing() {
  const { nodeID } = useLibraryPageContext();
  const tags = useWatch((s) => s.draft.tags);
  const {
    currentTagItems,
    queryResults,
    isSuggestEnabled,
    loadingTags,
    handleQuery,
    handleSuggestTags,
    handleChange,
  } = useLibraryPageTagsBlockEditing();

  return (
    <VStack w="full" gap="2" alignItems="start">
      <HStack w="full" gap="1" alignItems="start">
        <MultiSelectPicker
          value={currentTagItems}
          onChange={handleChange}
          onQuery={handleQuery}
          queryResults={queryResults}
          allowNewValues={true}
          inputPlaceholder="Add tags..."
          autoColour={true}
          size="sm"
        />
        {isSuggestEnabled && (
          <IntelligenceAction
            title="Suggest tags for this page"
            onClick={handleSuggestTags}
            variant="subtle"
            loading={loadingTags}
          />
        )}
      </HStack>
      <TaggedItemsForTags tags={tags} nodeID={nodeID} />
    </VStack>
  );
}

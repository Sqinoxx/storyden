"use client";

import {
  TreeView as ArkTreeView,
  createTreeCollection,
} from "@ark-ui/react/tree-view";
import { useMemo, useState } from "react";

import { BulletIcon } from "@/components/ui/icons/Bullet";
import { ChevronRightIcon } from "@/components/ui/icons/Chevron";
import { useTranslation } from "@/lib/i18n";
import { CategoryTree, findCategoryPath } from "@/lib/category/tree";
import { css, cx } from "@/styled-system/css";
import { styled } from "@/styled-system/jsx";
import { treeView } from "@/styled-system/recipes";

export type EmptyOption = {
  label: string;
  value: string;
};

export type CategoryTreePanelProps = {
  tree: CategoryTree[];
  value?: string;
  isSelectable: (node: CategoryTree) => boolean;
  onSelect: (value: string) => void;
  emptyOption?: EmptyOption;
};

const touchTargetStyles = css({
  "& [data-part='branch-control']": {
    h: { base: "10", md: "8" },
  },
});

const emptyOptionStyles = css({
  alignItems: "center",
  borderRadius: "l2",
  color: "fg.muted",
  cursor: "pointer",
  display: "flex",
  fontWeight: "medium",
  h: { base: "10", md: "8" },
  px: "2",
  textStyle: "sm",
  w: "full",
  _hover: {
    background: "bg.emphasized",
    color: "fg.emphasized",
  },
  _selected: {
    background: "bg.selected",
    color: "fg.selected",
  },
});

export function CategoryTreePanel({
  tree,
  value,
  isSelectable,
  onSelect,
  emptyOption,
}: CategoryTreePanelProps) {
  const t = useTranslation();

  const [expandedValue, setExpandedValue] = useState<string[]>(() =>
    findCategoryPath(tree, value).map((node) => node.id),
  );

  const collection = useMemo(
    () =>
      createTreeCollection<CategoryTree>({
        nodeToValue: (cat) => cat?.id ?? "",
        nodeToString: (cat) => cat?.name ?? "",
        nodeToChildren: (cat) => cat?.children ?? [],
        rootNode: { children: tree } as CategoryTree,
      }),
    [tree],
  );

  const styles = treeView();

  function toggle(id: string) {
    setExpandedValue((prev) =>
      prev.includes(id) ? prev.filter((v) => v !== id) : [...prev, id],
    );
  }

  return (
    <styled.div w="full" display="flex" flexDirection="column" gap="1">
      {emptyOption && (
        <styled.button
          type="button"
          className={emptyOptionStyles}
          data-selected={value === emptyOption.value ? "" : undefined}
          onClick={() => onSelect(emptyOption.value)}
        >
          {emptyOption.label}
        </styled.button>
      )}

      <ArkTreeView.Root
        className={cx(styles.root, touchTargetStyles)}
        collection={collection}
        expandOnClick={false}
        expandedValue={expandedValue}
        onExpandedChange={(details) => setExpandedValue(details.expandedValue)}
        selectedValue={value ? [value] : []}
      >
        <ArkTreeView.Tree className={styles.tree}>
          {collection.rootNode.children.map((node, index) => (
            <CategoryTreeNode
              key={node.id}
              node={node}
              indexPath={[index]}
              styles={styles}
              isSelectable={isSelectable}
              onSelect={onSelect}
              onToggle={toggle}
              notSelectableHint={t.category.categoryNotSelectable}
            />
          ))}
        </ArkTreeView.Tree>
      </ArkTreeView.Root>
    </styled.div>
  );
}

type NodeProps = {
  node: CategoryTree;
  indexPath: number[];
  styles: ReturnType<typeof treeView>;
  isSelectable: (node: CategoryTree) => boolean;
  onSelect: (value: string) => void;
  onToggle: (id: string) => void;
  notSelectableHint: string;
};

function CategoryTreeNode({
  node,
  indexPath,
  styles,
  isSelectable,
  onSelect,
  onToggle,
  notSelectableHint,
}: NodeProps) {
  const hasChildren = node.children.length > 0;
  const selectable = isSelectable(node);

  function handleClick() {
    if (selectable) {
      onSelect(node.id);
      return;
    }

    onToggle(node.id);
  }

  return (
    <ArkTreeView.NodeProvider node={node} indexPath={indexPath}>
      <ArkTreeView.Branch className={styles.branch}>
        <ArkTreeView.BranchControl className={styles.branchControl}>
          {hasChildren ? (
            <ArkTreeView.BranchTrigger className={styles.branchIndicator}>
              <ChevronRightIcon />
            </ArkTreeView.BranchTrigger>
          ) : (
            <styled.span className={styles.branchIndicator}>
              <BulletIcon />
            </styled.span>
          )}

          <ArkTreeView.BranchText asChild className={styles.branchText}>
            <styled.button
              type="button"
              title={selectable ? undefined : notSelectableHint}
              flex="1"
              textAlign="left"
              color={selectable ? undefined : "fg.subtle"}
              onClick={handleClick}
            >
              {node.name}
            </styled.button>
          </ArkTreeView.BranchText>
        </ArkTreeView.BranchControl>

        <ArkTreeView.BranchContent className={styles.branchContent}>
          {node.children.map((child, index) => (
            <CategoryTreeNode
              key={child.id}
              node={child}
              indexPath={[...indexPath, index]}
              styles={styles}
              isSelectable={isSelectable}
              onSelect={onSelect}
              onToggle={onToggle}
              notSelectableHint={notSelectableHint}
            />
          ))}
        </ArkTreeView.BranchContent>
      </ArkTreeView.Branch>
    </ArkTreeView.NodeProvider>
  );
}

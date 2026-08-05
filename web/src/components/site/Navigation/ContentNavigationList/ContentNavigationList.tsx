"use client";

import { CategoryListOKResponse, NodeListResult, Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { CategoryList } from "@/components/category/CategoryList/CategoryList";
import { LStack, styled } from "@/styled-system/jsx";
import { hasPermission, isModeratorOrAdmin } from "@/utils/permissions";

import { CollectionsAnchor } from "../Anchors/Collections";
import { LinksAnchor } from "../Anchors/Link";
import { MembersAnchor } from "../Anchors/Members";
import { RolesAnchor } from "../Anchors/Roles";
import { TagsAnchor } from "../Anchors/Tags";
import { DraftsSidebarSection } from "../DraftsSidebarSection/DraftsSidebarSection";
import { LibraryNavigationTree } from "../LibraryNavigationTree/LibraryNavigationTree";
import { useNavigation } from "../useNavigation";

type Props = {
  initialNodeList?: NodeListResult;
  initialCategoryList?: CategoryListOKResponse;
};

export function ContentNavigationList(props: Props) {
  const { nodeSlug } = useNavigation();
  const session = useSession();
  const isStaff = isModeratorOrAdmin(session);
  const isAdmin = hasPermission(session, Permission.ADMINISTRATOR);

  return (
    <styled.nav
      display="flex"
      flexDir="column"
      gap="4"
      height="full"
      width="full"
      minH="0"
      alignItems="start"
      justifyContent="space-between"
    >
      <LStack
        gap="1"
        overflowY="scroll"
        style={{
          scrollbarWidth: "none",
        }}
      >
        <CategoryList initialCategoryList={props.initialCategoryList} />
        <styled.hr
          width="full"
          border="none"
          borderColor="border.subtle"
          my="1.5"
          style={{ borderTop: "1px solid", opacity: 0.15 }}
        />
        <LibraryNavigationTree
          initialNodeList={props.initialNodeList}
          currentNode={nodeSlug}
          visibility={["draft", "review", "unlisted", "published"]}
        />
        <DraftsSidebarSection />
      </LStack>

      <LStack gap="1">
        <CollectionsAnchor />
        {isAdmin && <LinksAnchor />}
        {isStaff && (
          <>
            <MembersAnchor />
            <RolesAnchor />
            <TagsAnchor />
          </>
        )}
      </LStack>
    </styled.nav>
  );
}

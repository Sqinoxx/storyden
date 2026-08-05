"use client";

import { useState } from "react";
import { useSWRConfig } from "swr";
import { PencilIcon, PlusIcon, TrashIcon } from "lucide-react";

import {
  getTagListKey,
  tagCreate,
  tagDelete,
  tagUpdate,
  useTagList,
} from "@/api/openapi-client/tags";
import { TagListResult, TagReference } from "@/api/openapi-schema";
import { TagBadge } from "@/components/tag/TagBadge";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Text } from "@/components/ui/text";
import { useTranslation } from "@/lib/i18n";
import { Box, Flex, HStack, LStack, styled } from "@/styled-system/jsx";

type Props = {
  initialTagList: TagListResult;
};

export function TagManagementScreen({ initialTagList }: Props) {
  const t = useTranslation();
  const { mutate } = useSWRConfig();
  const { data } = useTagList(undefined, { swr: { fallbackData: initialTagList } });

  const [search, setSearch] = useState("");
  const [newTagName, setNewTagName] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const [tagToDelete, setTagToDelete] = useState<TagReference | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const [tagToEdit, setTagToEdit] = useState<TagReference | null>(null);
  const [editTagName, setEditTagName] = useState("");
  const [isEditing, setIsEditing] = useState(false);
  const [editError, setEditError] = useState<string | null>(null);

  const tags = data?.tags ?? [];
  const filteredTags = tags.filter((tag) =>
    tag.name.toLowerCase().includes(search.toLowerCase().trim()),
  );

  const totalItemsCount = tags.reduce((sum, tag) => sum + tag.item_count, 0);

  async function handleCreateTag(e: React.FormEvent) {
    e.preventDefault();
    const name = newTagName.trim();
    if (!name) return;

    setIsCreating(true);
    setCreateError(null);

    try {
      await tagCreate({ name });
      setNewTagName("");
      await mutate(getTagListKey());
    } catch (err: any) {
      setCreateError(err?.message || "Fehler beim Erstellen des Tags.");
    } finally {
      setIsCreating(false);
    }
  }

  async function handleDeleteTag() {
    if (!tagToDelete) return;

    setIsDeleting(true);
    try {
      await tagDelete(tagToDelete.name);
      setTagToDelete(null);
      await mutate(getTagListKey());
    } catch (err) {
      console.error(err);
    } finally {
      setIsDeleting(false);
    }
  }

  function handleOpenEdit(tag: TagReference) {
    setTagToEdit(tag);
    setEditTagName(tag.name);
    setEditError(null);
  }

  async function handleUpdateTag(e: React.FormEvent) {
    e.preventDefault();
    if (!tagToEdit) return;
    const newName = editTagName.trim();
    if (!newName) return;

    setIsEditing(true);
    setEditError(null);

    try {
      await tagUpdate(tagToEdit.name, { name: newName });
      setTagToEdit(null);
      await mutate(getTagListKey());
    } catch (err: any) {
      setEditError(err?.message || "Fehler beim Aktualisieren des Tags.");
    } finally {
      setIsEditing(false);
    }
  }

  return (
    <LStack gap="6" width="full">
      {/* Header & Breadcrumbs */}
      <LStack gap="2">
        <Breadcrumbs
          index={{ label: t.nav.admin, href: "/admin" }}
          crumbs={[{ label: t.tags.managementTitle, href: "/admin/tags" }]}
        />
        <styled.h1 textStyle="2xl" fontWeight="bold">
          {t.tags.managementTitle}
        </styled.h1>
        <Text textStyle="sm" color="fg.muted">
          {t.tags.managementSubtitle}
        </Text>
      </LStack>

      {/* Stats Summary Cards */}
      <HStack gap="4" width="full">
        <Box
          flex="1"
          p="4"
          borderRadius="l2"
          borderWidth="thin"
          borderColor="border.subtle"
          bg="bg.default"
        >
          <Text textStyle="xs" color="fg.muted" fontWeight="medium">
            {t.tags.totalTags}
          </Text>
          <Text textStyle="2xl" fontWeight="bold" mt="1">
            {tags.length}
          </Text>
        </Box>
        <Box
          flex="1"
          p="4"
          borderRadius="l2"
          borderWidth="thin"
          borderColor="border.subtle"
          bg="bg.default"
        >
          <Text textStyle="xs" color="fg.muted" fontWeight="medium">
            {t.tags.totalTaggedItems}
          </Text>
          <Text textStyle="2xl" fontWeight="bold" mt="1">
            {totalItemsCount}
          </Text>
        </Box>
      </HStack>

      {/* Create New Tag Card */}
      <Box
        p="5"
        borderRadius="l2"
        borderWidth="thin"
        borderColor="border.default"
        bg="bg.default"
      >
        <styled.form onSubmit={handleCreateTag}>
          <LStack gap="3">
            <styled.h2 textStyle="md" fontWeight="semibold">
              {t.tags.createTitle}
            </styled.h2>
            <HStack gap="3">
              <Input
                placeholder={t.tags.createPlaceholder}
                value={newTagName}
                onChange={(e) => setNewTagName(e.target.value)}
                disabled={isCreating}
                flex="1"
              />
              <Button
                type="submit"
                variant="solid"
                disabled={!newTagName.trim() || isCreating}
                loading={isCreating}
                loadingText={t.tags.creating}
              >
                <PlusIcon size={16} />
                {t.tags.createButton}
              </Button>
            </HStack>
            {createError && (
              <Text textStyle="xs" color="fg.error">
                {createError}
              </Text>
            )}
          </LStack>
        </styled.form>
      </Box>

      {/* Search & Filter Bar */}
      <Flex justifyContent="space-between" alignItems="center" gap="4" width="full">
        <HStack flex="1" maxW="md">
          <Input
            placeholder={t.tags.searchPlaceholder}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            width="full"
          />
        </HStack>
      </Flex>

      {/* Tag Grid List */}
      {filteredTags.length === 0 ? (
        <Box
          py="12"
          px="4"
          textAlign="center"
          borderWidth="thin"
          borderStyle="dashed"
          borderColor="border.subtle"
          borderRadius="l2"
        >
          <Text color="fg.muted">{t.tags.emptyState}</Text>
        </Box>
      ) : (
        <styled.div
          display="grid"
          gridTemplateColumns={{
            base: "1fr",
            sm: "repeat(2, 1fr)",
            md: "repeat(3, 1fr)",
          }}
          gap="3"
          width="full"
        >
          {filteredTags.map((tag) => (
            <Flex
              key={tag.name}
              p="3"
              alignItems="center"
              justifyContent="space-between"
              borderWidth="thin"
              borderColor="border.subtle"
              borderRadius="l2"
              bg="bg.default"
              _hover={{ borderColor: "border.default" }}
              style={{ transition: "all 0.15s ease" }}
            >
              <HStack gap="2" overflow="hidden">
                <TagBadge tag={tag} showItemCount />
              </HStack>

              <HStack gap="1">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleOpenEdit(tag)}
                  title={t.tags.editButton}
                  color="fg.muted"
                  _hover={{ color: "fg.default", bg: "bg.muted" }}
                >
                  <PencilIcon size={16} />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setTagToDelete(tag)}
                  title={t.tags.deleteButton}
                  color="fg.muted"
                  _hover={{ color: "fg.error", bg: "bg.muted" }}
                >
                  <TrashIcon size={16} />
                </Button>
              </HStack>
            </Flex>
          ))}
        </styled.div>
      )}

      {/* Edit Tag Modal / Backdrop */}
      {tagToEdit && (
        <styled.div
          position="fixed"
          inset="0"
          style={{
            zIndex: 100,
            backgroundColor: "rgba(0, 0, 0, 0.6)",
            backdropFilter: "blur(4px)",
          }}
          display="flex"
          alignItems="center"
          justifyContent="center"
          p="4"
        >
          <Box
            bg="bg.default"
            p="6"
            borderRadius="l3"
            borderWidth="thin"
            borderColor="border.default"
            maxW="md"
            width="full"
            shadow="xl"
          >
            <styled.form onSubmit={handleUpdateTag}>
              <LStack gap="4">
                <styled.h3 textStyle="lg" fontWeight="bold">
                  {t.tags.editTitle}
                </styled.h3>
                <Input
                  placeholder={t.tags.editPlaceholder}
                  value={editTagName}
                  onChange={(e) => setEditTagName(e.target.value)}
                  disabled={isEditing}
                  width="full"
                  autoFocus
                />
                {editError && (
                  <Text textStyle="xs" color="fg.error">
                    {editError}
                  </Text>
                )}
                <HStack justifyContent="flex-end" gap="3" mt="2">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => setTagToEdit(null)}
                    disabled={isEditing}
                  >
                    {t.tags.cancel}
                  </Button>
                  <Button
                    type="submit"
                    variant="solid"
                    disabled={!editTagName.trim() || isEditing}
                    loading={isEditing}
                    loadingText={t.tags.saving}
                  >
                    {t.tags.save}
                  </Button>
                </HStack>
              </LStack>
            </styled.form>
          </Box>
        </styled.div>
      )}

      {/* Delete Confirmation Modal / Backdrop */}
      {tagToDelete && (
        <styled.div
          position="fixed"
          inset="0"
          style={{
            zIndex: 100,
            backgroundColor: "rgba(0, 0, 0, 0.6)",
            backdropFilter: "blur(4px)",
          }}
          display="flex"
          alignItems="center"
          justifyContent="center"
          p="4"
        >
          <Box
            bg="bg.default"
            p="6"
            borderRadius="l3"
            borderWidth="thin"
            borderColor="border.default"
            maxW="md"
            width="full"
            shadow="xl"
          >
            <LStack gap="4">
              <styled.h3 textStyle="lg" fontWeight="bold">
                {t.tags.deleteTitle}
              </styled.h3>
              <Text textStyle="sm" color="fg.muted">
                {t.tags.deleteConfirmation.replace("{name}", tagToDelete.name)}
              </Text>
              <HStack justifyContent="flex-end" gap="3" mt="2">
                <Button
                  variant="outline"
                  onClick={() => setTagToDelete(null)}
                  disabled={isDeleting}
                >
                  {t.tags.cancel}
                </Button>
                <Button
                  variant="solid"
                  onClick={handleDeleteTag}
                  loading={isDeleting}
                  loadingText={t.tags.deleting}
                  style={{
                    backgroundColor: "var(--colors-red-600, #dc2626)",
                    color: "white",
                  }}
                >
                  {t.tags.deleteButton}
                </Button>
              </HStack>
            </LStack>
          </Box>
        </styled.div>
      )}
    </LStack>
  );
}

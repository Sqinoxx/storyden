"use client";

import { useMemo, useState } from "react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import {
  accessPassCreate,
  accessPassDelete,
  accessPassUpdate,
} from "@/api/openapi-client/access";
import { useCategoryList } from "@/api/openapi-client/categories";
import { useDriveFolderList } from "@/api/openapi-client/drive";
import { AccessPass } from "@/api/openapi-schema";
import { DeleteWithConfirmationButton } from "@/components/site/DeleteConfirmationButton";
import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { FormControl } from "@/components/ui/FormControl";
import { FormHelperText } from "@/components/ui/FormHelperText";
import { FormLabel } from "@/components/ui/FormLabel";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { useAccessRevalidate } from "@/lib/access/useAccessRevalidate";
import { CategoryTree, buildCategoryTree } from "@/lib/category/tree";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";

type Props = {
  isOpen: boolean;
  onClose: () => void;
  pass?: AccessPass;
};

export function PassEditor(props: Props) {
  if (!props.isOpen) return null;
  return <PassEditorContent {...props} />;
}

const textareaStyle = {
  width: "full",
  borderWidth: "thin",
  borderRadius: "sm",
  borderColor: "border.default",
  padding: "2",
} as const;

function flatten(
  nodes: CategoryTree[],
  depth = 0,
): { node: CategoryTree; depth: number }[] {
  return nodes.flatMap((node) => [
    { node, depth },
    ...flatten(node.children, depth + 1),
  ]);
}

function PassEditorContent({ isOpen, onClose, pass }: Props) {
  const t = useTranslation();
  const m = t.access.manage;
  const revalidate = useAccessRevalidate();
  const { data: categoryData } = useCategoryList();
  const { data: driveData } = useDriveFolderList();

  const [name, setName] = useState(pass?.name ?? "");
  const [description, setDescription] = useState(pass?.description ?? "");
  const [priceLabel, setPriceLabel] = useState(pass?.price_label ?? "");
  const [paymentInfo, setPaymentInfo] = useState(pass?.payment_info ?? "");
  const [requestable, setRequestable] = useState(pass?.requestable ?? true);
  const [categoryIDs, setCategoryIDs] = useState<Set<string>>(
    () => new Set(pass?.category_ids ?? []),
  );
  const [driveFolderIDs, setDriveFolderIDs] = useState<Set<string>>(
    () => new Set(pass?.drive_folder_ids ?? []),
  );
  const [busy, setBusy] = useState(false);

  const rows = useMemo(
    () => flatten(buildCategoryTree(categoryData?.categories ?? [])),
    [categoryData],
  );

  // A ticked parent already covers its subtree, so its descendants are shown
  // as included rather than offered as separate choices.
  const coveredByParent = useMemo(() => {
    const covered = new Set<string>();
    const mark = (n: CategoryTree, inherited: boolean) => {
      if (inherited) covered.add(n.id);
      n.children.forEach((c) => mark(c, inherited || categoryIDs.has(n.id)));
    };
    buildCategoryTree(categoryData?.categories ?? []).forEach((n) =>
      mark(n, false),
    );
    return covered;
  }, [categoryData, categoryIDs]);

  function toggle(id: string, checked: boolean) {
    setCategoryIDs((prev) => {
      const next = new Set(prev);
      if (checked) next.add(id);
      else next.delete(id);
      return next;
    });
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    const body = {
      name: name.trim(),
      description,
      price_label: priceLabel,
      payment_info: paymentInfo,
      requestable,
      category_ids: [...categoryIDs].filter((id) => !coveredByParent.has(id)),
      drive_folder_ids: [...driveFolderIDs],
    };

    setBusy(true);
    await handle(
      async () => {
        if (pass) {
          await accessPassUpdate(pass.id, body);
        } else {
          await accessPassCreate(body);
        }
        await revalidate();
        toast.success(m.passSaved);
        onClose();
      },
      { cleanup: async () => setBusy(false) },
    );
  }

  async function onDelete() {
    if (!pass) return;
    await handle(async () => {
      await accessPassDelete(pass.id);
      await revalidate();
      toast.success(m.passDeleted);
      onClose();
    });
  }

  return (
    <ModalDrawer
      isOpen={isOpen}
      onClose={onClose}
      title={pass ? m.editPass : m.createPass}
    >
      <styled.form
        display="flex"
        flexDir="column"
        gap="4"
        w="full"
        onSubmit={onSubmit}
        data-testid="access-pass-editor"
      >
        <FormControl>
          <FormLabel>{m.fieldName}</FormLabel>
          <Input
            aria-label={m.fieldName}
            value={name}
            onChange={(e) => setName(e.currentTarget.value)}
            required
          />
        </FormControl>

        <FormControl>
          <FormLabel>{m.fieldPrice}</FormLabel>
          <Input
            aria-label={m.fieldPrice}
            value={priceLabel}
            onChange={(e) => setPriceLabel(e.currentTarget.value)}
          />
          <FormHelperText>{m.fieldPriceHint}</FormHelperText>
        </FormControl>

        <FormControl>
          <FormLabel>{m.fieldDescription}</FormLabel>
          <styled.textarea
            {...textareaStyle}
            aria-label={m.fieldDescription}
            rows={2}
            value={description}
            onChange={(e) => setDescription(e.currentTarget.value)}
          />
        </FormControl>

        <FormControl>
          <FormLabel>{m.fieldPayment}</FormLabel>
          <styled.textarea
            {...textareaStyle}
            aria-label={m.fieldPayment}
            rows={3}
            value={paymentInfo}
            onChange={(e) => setPaymentInfo(e.currentTarget.value)}
          />
          <FormHelperText>{m.fieldPaymentHint}</FormHelperText>
        </FormControl>

        <FormControl>
          <FormLabel>{m.fieldCategories}</FormLabel>
          <FormHelperText>{m.fieldCategoriesHint}</FormHelperText>
          <LStack
            gap="1"
            maxH="64"
            overflowY="auto"
            borderWidth="thin"
            borderColor="border.default"
            borderRadius="sm"
            p="2"
          >
            {rows.map(({ node, depth }) => {
              const inherited = coveredByParent.has(node.id);
              return (
                <HStack key={node.id} style={{ paddingLeft: `${depth}rem` }}>
                  <Checkbox
                    size="sm"
                    checked={inherited || categoryIDs.has(node.id)}
                    disabled={inherited}
                    onCheckedChange={({ checked }) =>
                      toggle(node.id, checked === true)
                    }
                  >
                    {node.name}
                  </Checkbox>
                </HStack>
              );
            })}
          </LStack>
        </FormControl>

        {(driveData?.folders.length ?? 0) > 0 && (
          <FormControl>
            <FormLabel>{m.fieldDriveFolders}</FormLabel>
            <FormHelperText>{m.fieldDriveFoldersHint}</FormHelperText>
            <LStack gap="1">
              {driveData?.folders.map((f) => (
                <Checkbox
                  key={f.id}
                  size="sm"
                  checked={driveFolderIDs.has(f.id)}
                  onCheckedChange={({ checked }) =>
                    setDriveFolderIDs((prev) => {
                      const next = new Set(prev);
                      if (checked === true) next.add(f.id);
                      else next.delete(f.id);
                      return next;
                    })
                  }
                >
                  {f.name}
                </Checkbox>
              ))}
            </LStack>
          </FormControl>
        )}

        <Checkbox
          size="sm"
          checked={requestable}
          onCheckedChange={({ checked }) => setRequestable(checked === true)}
        >
          {m.fieldRequestable}
        </Checkbox>

        <HStack gap="2" w="full">
          {pass && (
            <DeleteWithConfirmationButton onDelete={onDelete}>
              {m.deletePass}
            </DeleteWithConfirmationButton>
          )}
          <Button type="submit" flex="1" loading={busy} disabled={!name.trim()}>
            {t.common.save}
          </Button>
        </HStack>
      </styled.form>
    </ModalDrawer>
  );
}

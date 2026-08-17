"use client";

import { useState } from "react";

import { handle } from "@/api/client";
import {
  getInvitationListKey,
  invitationCreate,
  invitationDelete,
  useInvitationList,
} from "@/api/openapi-client/invitations";
import { Identifier } from "@/api/openapi-schema";
import { useSWRConfig } from "swr";

export function useInvitationsSettings() {
  const { mutate } = useSWRConfig();
  const { data, error } = useInvitationList();
  const [createdID, setCreatedID] = useState<Identifier | undefined>();
  const [isCreating, setCreating] = useState(false);
  const [isMultiFormOpen, setMultiFormOpen] = useState(false);

  async function handleCreate() {
    if (isCreating) {
      return;
    }

    setCreating(true);

    await handle(
      async () => {
        const created = await invitationCreate({});
        setCreatedID(created.id);
        await mutate(getInvitationListKey());
      },
      {
        cleanup: async () => setCreating(false),
      },
    );
  }

  function handleOpenMultiForm() {
    setMultiFormOpen(true);
  }

  function handleCloseMultiForm() {
    setMultiFormOpen(false);
  }

  function handleMultiCreated(id: Identifier) {
    setCreatedID(id);
    setMultiFormOpen(false);
  }

  async function handleDelete(id: Identifier) {
    await handle(async () => {
      await invitationDelete(id);

      if (createdID === id) {
        setCreatedID(undefined);
      }

      await mutate(getInvitationListKey());
    });
  }

  if (!data) {
    return {
      ready: false as const,
      error,
    };
  }

  return {
    ready: true as const,
    invitations: data.invitations,
    createdID,
    isCreating,
    isMultiFormOpen,
    handleCreate,
    handleDelete,
    handleDismissCreated: () => setCreatedID(undefined),
    handleOpenMultiForm,
    handleCloseMultiForm,
    handleMultiCreated,
  };
}

"use client";

import React, { ReactNode } from "react";

import { ProfileReference } from "@/api/openapi-schema";
import { useAccessPasses } from "@/lib/access/useAccess";
import { useDisclosure } from "@/utils/useDisclosure";

import { GrantDialog } from "./GrantDialog";

type Props = {
  children: ReactNode;
  profile: ProfileReference;
};

// Lets a manager unlock someone straight from their profile menu, for the
// common case of a member paying in person without requesting first.
export function MemberAccessTrigger({ children, profile }: Props) {
  const { passes } = useAccessPasses();
  const { onOpen, onClose, isOpen } = useDisclosure();

  if (passes.length === 0) return null;

  const trigger = React.isValidElement<{ onClick?: React.MouseEventHandler }>(
    children,
  )
    ? React.cloneElement(children, { onClick: onOpen })
    : children;

  return (
    <>
      {trigger}
      <GrantDialog
        isOpen={isOpen}
        onClose={onClose}
        passes={passes}
        memberHandle={profile.handle}
        memberName={profile.name}
      />
    </>
  );
}

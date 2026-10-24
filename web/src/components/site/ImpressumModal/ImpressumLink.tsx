"use client";

import React, { useState } from "react";
import { useSession } from "@/auth";
import { styled } from "@/styled-system/jsx";
import { ImpressumModal } from "./ImpressumModal";

export function ImpressumLink() {
  const session = useSession();
  const [isOpen, setIsOpen] = useState(false);

  if (!session) {
    return null;
  }

  return (
    <>
      <styled.button
        type="button"
        onClick={() => setIsOpen(true)}
        color="fg.subtle"
        fontSize="xs"
        cursor="pointer"
        bg="transparent"
        border="none"
        p="0"
        transition="colors"
        _hover={{
          color: "fg.default",
          textDecoration: "underline",
        }}
      >
        Impressum
      </styled.button>

      <styled.span color="fg.subtle">•</styled.span>

      <ImpressumModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  );
}

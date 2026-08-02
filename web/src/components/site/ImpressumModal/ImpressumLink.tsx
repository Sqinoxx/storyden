"use client";

import React, { useState } from "react";
import { styled } from "@/styled-system/jsx";
import { ImpressumModal } from "./ImpressumModal";

export function ImpressumLink() {
  const [isOpen, setIsOpen] = useState(false);

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

      <ImpressumModal isOpen={isOpen} onClose={() => setIsOpen(false)} />
    </>
  );
}

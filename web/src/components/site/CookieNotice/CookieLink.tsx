"use client";

import React from "react";
import { styled } from "@/styled-system/jsx";
import { openCookieSettings } from "./CookieNotice";

export function CookieLink() {
  return (
    <styled.button
      type="button"
      onClick={openCookieSettings}
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
      Cookies
    </styled.button>
  );
}

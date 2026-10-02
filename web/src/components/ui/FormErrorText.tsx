"use client";

import { ComponentProps } from "react";

import { useTranslateMessage } from "@/lib/i18n";
import { type RecipeVariantProps, cva } from "@/styled-system/css";
import { styled } from "@/styled-system/jsx";

const formErrorText = cva({
  base: {
    color: "fg.destructive",
    fontSize: "xs",
  },
});

export type FormErrorTextVariants = RecipeVariantProps<typeof formErrorText>;

const StyledFormErrorText = styled("p", formErrorText);

export function FormErrorText({
  children,
  ...props
}: ComponentProps<typeof StyledFormErrorText>) {
  const translate = useTranslateMessage();

  return (
    <StyledFormErrorText {...props}>
      {typeof children === "string" ? translate(children) : children}
    </StyledFormErrorText>
  );
}

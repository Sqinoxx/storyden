"use client";

import { useState } from "react";

import { HideIcon } from "@/components/ui/icons/HideIcon";
import { ShowIcon } from "@/components/ui/icons/ShowIcon";
import { useTranslation } from "@/lib/i18n";

import { IconButton } from "./icon-button";
import { InputGroup } from "./input-group";
import { Input, InputProps } from "./input";

export type PasswordInputProps = Omit<InputProps, "type">;

export function PasswordInput(props: PasswordInputProps) {
  const t = useTranslation();
  const [visible, setVisible] = useState(false);

  const label = visible ? t.auth.hidePassword : t.auth.showPassword;

  return (
    <InputGroup
      size={props.size === "xs" || props.size === "sm" ? props.size : "md"}
      endElement={
        <IconButton
          type="button"
          variant="ghost"
          size="xs"
          aria-label={label}
          title={label}
          aria-pressed={visible}
          onClick={() => setVisible((v) => !v)}
        >
          {visible ? <HideIcon /> : <ShowIcon />}
        </IconButton>
      }
    >
      <Input type={visible ? "text" : "password"} {...props} />
    </InputGroup>
  );
}

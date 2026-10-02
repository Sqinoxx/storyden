"use client";

import type { KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { BiometricIcon } from "@/components/ui/icons/Biometric";
import { Input } from "@/components/ui/input";
import { Flex, styled } from "@/styled-system/jsx";

import { SemesterField } from "../SemesterField";
import { Props, useRegisterHandleForm } from "./useRegisterHandleForm";
import { useTranslateMessage, useTranslation } from "@/lib/i18n";

export function RegisterHandleForm(props: Props) {
  const t = useTranslation();
  const {
    form: {
      register,
      isWebauthnEnabled,
      handlePassword,
      handleWebauthn,
      errors,
      isSubmitting,
    },
  } = useRegisterHandleForm(props);
  const translate = useTranslateMessage();

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handlePassword(e);
    }
  };

  return (
    <styled.form
      w="full"
      display="flex"
      flexDir="column"
      gap="2"
      textAlign="center"
      onSubmit={handlePassword}
      onKeyDown={handleKeyDown}
    >
      <Input
        type="text"
        autoCapitalize="none"
        autoCorrect="off"
        autoComplete="username"
        w="full"
        size="sm"
        textAlign="center"
        placeholder={t.auth.usernamePlaceholder}
        required
        {...register("identifier")}
      />
      <styled.p color="fg.error" fontSize="sm">
        {errors.identifier?.message && translate(errors.identifier.message)}
      </styled.p>
      <SemesterField register={register("semester")} />
      <styled.p color="fg.error" fontSize="sm">
        {errors.semester?.message && translate(errors.semester.message)}
      </styled.p>
      <Flex alignItems="center" gap="2">
        <Input
          type="password"
          w="full"
          size="sm"
          textAlign="center"
          placeholder={t.auth.passwordLabel}
          autoComplete="new-password"
          {...register("token")}
        />
        {props.webauthn && isWebauthnEnabled && (
          <>
            <styled.span>{t.auth.or}</styled.span>

            <Button
              w="full"
              variant="ghost"
              size="sm"
              type="button"
              onClick={handleWebauthn}
            >
              <styled.span display="flex" gap="1" alignItems="center" px="4">
                {t.auth.device}
                <BiometricIcon />
              </styled.span>
            </Button>
          </>
        )}
      </Flex>
      <styled.p color="fg.error" fontSize="sm">
        {errors.token?.message && translate(errors.token.message)}
      </styled.p>
      <Button type="submit" w="full" loading={isSubmitting}>
        {t.auth.register}
      </Button>
      <styled.p color="fg.error" fontSize="sm">
        {errors.root?.message && translate(errors.root.message)}
      </styled.p>
    </styled.form>
  );
}

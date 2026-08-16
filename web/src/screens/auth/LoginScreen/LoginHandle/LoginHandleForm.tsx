"use client";

import type { KeyboardEvent } from "react";
import { Controller } from "react-hook-form";

import { Checkbox } from "@/components/ui/checkbox";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { Flex, styled } from "@/styled-system/jsx";

import { useLoginHandleForm } from "./useLoginHandleForm";

export function LoginHandleForm() {
  const t = useTranslation();
  const {
    form: {
      register,
      control,
      isWebauthnEnabled,
      handlePassword,
      handleWebauthn,
      errors,
      isSubmitting,
    },
  } = useLoginHandleForm();

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
        {errors.identifier?.message}
      </styled.p>
      <Flex alignItems="center" gap="2">
        <Input
          type="password"
          w="full"
          size="sm"
          textAlign="center"
          placeholder={t.auth.passwordPlaceholder}
          autoComplete="current-password"
          {...register("token")}
        />
      </Flex>
      <styled.p color="fg.error" fontSize="sm">
        {errors.token?.message}
      </styled.p>
      <Flex alignItems="center" justifyContent="center" gap="2">
        <Controller
          control={control}
          name="remember_me"
          render={({ field }) => (
            <Checkbox
              size="sm"
              checked={!!field.value}
              onCheckedChange={({ checked }) => {
                field.onChange(checked === true);
              }}
            >
              {t.auth.rememberMe}
            </Checkbox>
          )}
        />
      </Flex>
      <Button type="submit" w="full" loading={isSubmitting}>
        {t.auth.login}
      </Button>
      <styled.p color="fg.error" fontSize="sm">
        {errors.root?.message}
      </styled.p>
    </styled.form>
  );
}

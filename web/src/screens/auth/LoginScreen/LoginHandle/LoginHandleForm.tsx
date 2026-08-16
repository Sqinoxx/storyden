"use client";

import type { KeyboardEvent } from "react";
import { Controller } from "react-hook-form";

import { Checkbox } from "@/components/ui/checkbox";
import { FormControl } from "@/components/ui/FormControl";
import { FormLabel } from "@/components/ui/FormLabel";
import { Button } from "@/components/ui/button";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
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
      gap="4"
      onSubmit={handlePassword}
      onKeyDown={handleKeyDown}
    >
      <FormControl>
        <FormLabel>{t.auth.identifierLabel}</FormLabel>
        <Input
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          autoComplete="username"
          w="full"
          size="sm"
          placeholder={t.auth.identifierPlaceholder}
          required
          {...register("identifier")}
        />
        {errors.identifier?.message && (
          <FormErrorText>{errors.identifier.message}</FormErrorText>
        )}
      </FormControl>

      <FormControl>
        <FormLabel>{t.auth.passwordLabel}</FormLabel>
        <Input
          type="password"
          w="full"
          size="sm"
          placeholder={t.auth.passwordPlaceholder}
          autoComplete="current-password"
          {...register("token")}
        />
        {errors.token?.message && (
          <FormErrorText>{errors.token.message}</FormErrorText>
        )}
      </FormControl>

      <Flex alignItems="center" justifyContent="space-between" gap="2">
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

      <Button type="submit" w="full" size="sm" loading={isSubmitting}>
        {t.auth.login}
      </Button>

      {errors.root?.message && (
        <FormErrorText textAlign="center">
          {errors.root.message}
        </FormErrorText>
      )}
    </styled.form>
  );
}

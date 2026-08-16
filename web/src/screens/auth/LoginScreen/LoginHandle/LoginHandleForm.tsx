"use client";

import type { KeyboardEvent } from "react";

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
        placeholder="username"
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
          placeholder="password"
          autoComplete="current-password"
          {...register("token")}
        />
      </Flex>
      <styled.p color="fg.error" fontSize="sm">
        {errors.token?.message}
      </styled.p>
      <Flex alignItems="center" justifyContent="center" gap="2">
        <styled.input
          id="remember-me"
          type="checkbox"
          cursor="pointer"
          {...register("remember_me")}
        />
        <styled.label htmlFor="remember-me" fontSize="sm" cursor="pointer">
          {t.auth.rememberMe}
        </styled.label>
      </Flex>
      <Button type="submit" w="full" loading={isSubmitting}>
        Login
      </Button>
      <styled.p color="fg.error" fontSize="sm">
        {errors.root?.message}
      </styled.p>
    </styled.form>
  );
}

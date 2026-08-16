"use client";

import type { KeyboardEvent } from "react";
import { Controller } from "react-hook-form";

import { Checkbox } from "@/components/ui/checkbox";
import { FormControl } from "@/components/ui/FormControl";
import { Button } from "@/components/ui/button";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { HStack, styled } from "@/styled-system/jsx";
import { vstack } from "@/styled-system/patterns";

import { useLoginEmailForm } from "./useLoginEmailForm";

export function LoginEmailForm() {
  const t = useTranslation();
  const { form, handlers } = useLoginEmailForm();

  const handleKeyDown = (e: KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handlers.handleSubmit(e);
    }
  };

  return (
    <styled.form
      className={vstack()}
      w="full"
      gap="2"
      textAlign="center"
      onSubmit={handlers.handleSubmit}
      onKeyDown={handleKeyDown}
    >
      <FormControl>
        <Input
          type="text"
          autoCapitalize="none"
          autoCorrect="off"
          autoComplete="username"
          w="full"
          size="sm"
          textAlign="center"
          placeholder={t.auth.identifierPlaceholder}
          required
          {...form.register("identifier")}
        />
        <FormErrorText>
          {form.formState.errors["identifier"]?.message}
        </FormErrorText>
      </FormControl>

      <FormControl>
        <Input
          type="password"
          w="full"
          size="sm"
          textAlign="center"
          placeholder={t.auth.passwordPlaceholder}
          autoComplete="current-password"
          {...form.register("password")}
        />

        <FormErrorText>
          {form.formState.errors["password"]?.message}
        </FormErrorText>
      </FormControl>

      <HStack justifyContent="center" gap="2">
        <Controller
          control={form.control}
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
      </HStack>

      <Button type="submit" w="full" loading={form.formState.isSubmitting}>
        {t.auth.login}
      </Button>

      <FormErrorText>{form.formState.errors["root"]?.message}</FormErrorText>
    </styled.form>
  );
}

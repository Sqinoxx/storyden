"use client";

import type { KeyboardEvent } from "react";

import { FormControl } from "@/components/ui/FormControl";
import { Button } from "@/components/ui/button";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { Input } from "@/components/ui/input";
import { styled } from "@/styled-system/jsx";
import { vstack } from "@/styled-system/patterns";

import { SemesterField } from "../SemesterField";
import { useRegisterEmailForm } from "./useRegisterEmailForm";

type Props = {
  invitationID?: string;
};

export function RegisterEmailForm(props: Props) {
  const { form, handlers } = useRegisterEmailForm(props);

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
          type="email"
          autoCapitalize="none"
          autoCorrect="off"
          autoComplete="email"
          w="full"
          size="sm"
          textAlign="center"
          placeholder="email address"
          required
          {...form.register("email")}
        />
        <FormErrorText>{form.formState.errors["email"]?.message}</FormErrorText>
      </FormControl>

      <FormControl>
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
          {...form.register("handle")}
        />
        <FormErrorText>
          {form.formState.errors["handle"]?.message}
        </FormErrorText>
      </FormControl>

      <FormControl>
        <Input
          type="password"
          w="full"
          size="sm"
          textAlign="center"
          placeholder="password"
          autoComplete="new-password"
          {...form.register("password")}
        />

        <FormErrorText>
          {form.formState.errors["password"]?.message}
        </FormErrorText>
      </FormControl>

      <FormControl>
        <SemesterField register={form.register("semester")} />
        <FormErrorText>
          {form.formState.errors["semester"]?.message}
        </FormErrorText>
      </FormControl>

      <Button type="submit" w="full" loading={form.formState.isSubmitting}>
        Register
      </Button>

      <FormErrorText>{form.formState.errors["root"]?.message}</FormErrorText>
    </styled.form>
  );
}

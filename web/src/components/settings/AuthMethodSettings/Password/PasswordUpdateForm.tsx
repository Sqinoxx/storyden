import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { Heading } from "@/components/ui/heading";
import { Input } from "@/components/ui/input";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { CardBox, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { usePasswordUpdate } from "./usePasswordUpdate";

export function PasswordUpdateForm() {
  const { t } = useLanguage();
  const { form, handlePasswordChange } = usePasswordUpdate();

  return (
    <styled.form className={lstack()} gap="2" onSubmit={handlePasswordChange}>
      <Heading>{t.settings?.auth?.passwordTitle || "Password"}</Heading>

      <FormControl>
        <Input
          maxW="xs"
          type="password"
          autoComplete="current-password"
          placeholder={t.settings?.auth?.currentPasswordPlaceholder || "current password"}
          {...form.register("old")}
        />
        <FormErrorText>{form.formState.errors["old"]?.message}</FormErrorText>
      </FormControl>

      <FormControl>
        <Input
          maxW="xs"
          type="password"
          autoComplete="new-password"
          placeholder={t.settings?.auth?.newPasswordPlaceholder || "new password"}
          {...form.register("new")}
        />
        <FormErrorText>{form.formState.errors["new"]?.message}</FormErrorText>
        <FormErrorText>{form.formState.errors["root"]?.message}</FormErrorText>
      </FormControl>

      <Button type="submit" variant="subtle" size="sm">
        {t.settings?.auth?.changePasswordButton || "Change password"}
      </Button>
    </styled.form>
  );
}

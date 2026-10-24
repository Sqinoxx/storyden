import { AccountSettings } from "@/components/settings/AccountSettings/AccountSettings";
import { PhoneSettings } from "@/components/settings/PhoneSettings/PhoneSettings";
import { LStack } from "@/styled-system/jsx";

export function MemberAccountSettingsScreen() {
  return (
    <LStack gap="4">
      <PhoneSettings />
      <AccountSettings />
    </LStack>
  );
}

import { useSession } from "@/auth";
import { SessionSettings } from "@/components/settings/SessionSettings/SessionSettings";

export function MemberSessionSettingsScreen() {
  const session = useSession();
  if (!session) {
    return null;
  }

  return <SessionSettings session={session} />;
}

import { getServerSession } from "@/auth/server-session";
import { getSettings } from "@/lib/settings/settings-server";
import { AskScreen } from "@/screens/ask/AskScreen";

export default async function Page() {
  const [session, settings] = await Promise.all([
    getServerSession(),
    getSettings(),
  ]);

  return <AskScreen session={session} initialSettings={settings} />;
}

import { getServerSession } from "@/auth/server-session";
import {
  UnauthenticatedBanner,
  UnreadyBanner,
} from "@/components/site/Unready";
import { getSettings } from "@/lib/settings/settings-server";
import { MySuggestionsScreen } from "@/screens/suggestions/MySuggestionsScreen";

export default async function Page() {
  try {
    const [session, settings] = await Promise.all([
      getServerSession(),
      getSettings(),
    ]);

    if (!session) {
      return <UnauthenticatedBanner initialSettings={settings} />;
    }

    return <MySuggestionsScreen />;
  } catch (e) {
    return <UnreadyBanner error={e} />;
  }
}

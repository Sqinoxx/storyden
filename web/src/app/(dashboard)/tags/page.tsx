import { tagList } from "@/api/openapi-server/tags";
import { getServerSession } from "@/auth/server-session";
import {
  UnauthenticatedBanner,
  UnreadyBanner,
} from "@/components/site/Unready";
import { getSettings } from "@/lib/settings/settings-server";
import { TagsIndexScreen } from "@/screens/tags/TagsIndexScreen";

export default async function Page() {
  try {
    const [session, settings] = await Promise.all([
      getServerSession(),
      getSettings(),
    ]);

    if (!session) {
      return <UnauthenticatedBanner initialSettings={settings} />;
    }

    const { data } = await tagList();
    return <TagsIndexScreen initialTagList={data} />;
  } catch (e) {
    return <UnreadyBanner error={e} />;
  }
}

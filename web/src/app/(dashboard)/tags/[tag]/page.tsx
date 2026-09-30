import { tagGet } from "@/api/openapi-server/tags";
import { getServerSession } from "@/auth/server-session";
import {
  UnauthenticatedBanner,
  UnreadyBanner,
} from "@/components/site/Unready";
import { getSettings } from "@/lib/settings/settings-server";
import { TagScreen } from "@/screens/tags/TagScreen";

type Props = {
  params: Promise<{
    tag: string;
  }>;
};

export default async function Page(props: Props) {
  const params = await props.params;
  try {
    const [session, settings] = await Promise.all([
      getServerSession(),
      getSettings(),
    ]);

    if (!session) {
      return <UnauthenticatedBanner initialSettings={settings} />;
    }

    const tag = decodeURIComponent(params.tag);

    const { data } = await tagGet(tag);
    return <TagScreen initialTag={data} slug={tag} />;
  } catch (e) {
    return <UnreadyBanner error={e} />;
  }
}

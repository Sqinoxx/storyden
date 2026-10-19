import { listingGet } from "@/api/openapi-server/listings";
import { getServerSession } from "@/auth/server-session";
import {
  UnauthenticatedBanner,
  UnreadyBanner,
} from "@/components/site/Unready";
import { getSettings } from "@/lib/settings/settings-server";
import { ListingScreen } from "@/screens/marketplace/ListingScreen";

type Props = {
  params: Promise<{ slug: string }>;
};

export default async function Page(props: Props) {
  const { slug } = await props.params;

  try {
    const [session, settings] = await Promise.all([
      getServerSession(),
      getSettings(),
    ]);

    if (!session) {
      return <UnauthenticatedBanner initialSettings={settings} />;
    }

    const { data } = await listingGet(slug, {
      cache: "no-store",
      next: { revalidate: 0 },
    });

    return <ListingScreen slug={slug} initialListing={data} />;
  } catch (e) {
    return <UnreadyBanner error={e} />;
  }
}

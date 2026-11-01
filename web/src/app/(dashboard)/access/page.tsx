import { accessPassList } from "@/api/openapi-server/access";
import { UnreadyBanner } from "@/components/site/Unready";
import { AccessScreen } from "@/screens/access/AccessScreen";

export default async function Page() {
  try {
    const { data } = await accessPassList({ cache: "no-store" });

    return <AccessScreen initialPasses={data} />;
  } catch (e) {
    return <UnreadyBanner error={e} />;
  }
}

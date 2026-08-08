import { driveFolderList } from "@/api/openapi-server/drive";
import { UnreadyBanner } from "@/components/site/Unready";
import { DriveIndexScreen } from "@/screens/drive/DriveIndexScreen";

export default async function Page() {
  try {
    const { data } = await driveFolderList({ cache: "no-store" });

    return <DriveIndexScreen folders={data} />;
  } catch (e) {
    return <UnreadyBanner error={e} />;
  }
}

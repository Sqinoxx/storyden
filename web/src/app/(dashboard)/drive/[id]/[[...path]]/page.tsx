import { driveFolderContents } from "@/api/openapi-server/drive";
import { UnreadyBanner } from "@/components/site/Unready";
import { DriveFolderScreen } from "@/screens/drive/DriveFolderScreen";

type Props = {
  params: Promise<{ id: string; path?: string[] }>;
};

export default async function Page(props: Props) {
  const { id, path } = await props.params;

  // Only the deepest segment identifies the folder to list. The trail above it
  // comes back with the response, so the URL does not have to carry it.
  const childID = path?.length
    ? decodeURIComponent(path[path.length - 1]!)
    : undefined;

  try {
    const { data } = await driveFolderContents(
      id,
      { child_id: childID },
      { cache: "no-store" },
    );

    return (
      <DriveFolderScreen folderID={id} childID={childID} contents={data} />
    );
  } catch (e) {
    return <UnreadyBanner error={e} />;
  }
}

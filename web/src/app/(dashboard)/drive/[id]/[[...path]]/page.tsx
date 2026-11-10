import {
  driveFolderContents,
  driveFolderList,
} from "@/api/openapi-server/drive";
import { UnreadyBanner } from "@/components/site/Unready";
import { DriveFolderScreen } from "@/screens/drive/DriveFolderScreen";
import { DriveLockedScreen } from "@/screens/drive/DriveLockedScreen";

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
    // A folder behind an access pass refuses its contents, but the folder
    // list still describes it, so members get the access card instead of an
    // error.
    const locked = await lockedFolder(id);
    if (locked) {
      return <DriveLockedScreen folder={locked} />;
    }
    return <UnreadyBanner error={e} />;
  }
}

async function lockedFolder(id: string) {
  try {
    const { data } = await driveFolderList({ cache: "no-store" });
    const folder = data.folders.find((f) => f.id === id);
    return folder?.access?.locked ? folder : undefined;
  } catch {
    return undefined;
  }
}

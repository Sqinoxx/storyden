import { useDriveFolderContents } from "@/api/openapi-client/drive";
import { DriveFolderContents } from "@/api/openapi-schema";
import { useSession } from "@/auth";

export type Props = {
  folderID: string;
  childID?: string;
  contents: DriveFolderContents;
};

export function useDriveFolderScreen(props: Props) {
  const session = useSession();

  const { data, error, mutate } = useDriveFolderContents(
    props.folderID,
    { child_id: props.childID },
    { swr: { fallbackData: props.contents } },
  );

  if (!data) {
    return {
      ready: false as const,
      error,
    };
  }

  return {
    ready: true as const,
    data,
    mutate,
    session,
  };
}

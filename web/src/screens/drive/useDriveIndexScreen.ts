import { useDriveFolderList } from "@/api/openapi-client/drive";
import { DriveFolderListOKResponse } from "@/api/openapi-schema";
import { useSession } from "@/auth";

export type Props = {
  folders: DriveFolderListOKResponse;
};

export function useDriveIndexScreen(props: Props) {
  const session = useSession();

  const { data, error, mutate } = useDriveFolderList({
    swr: { fallbackData: props.folders },
  });

  if (!data) {
    return {
      ready: false as const,
      error,
    };
  }

  return {
    ready: true as const,
    data: {
      folders: data.folders,
    },
    mutate,
    session,
  };
}

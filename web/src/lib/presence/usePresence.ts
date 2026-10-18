import useSWR from "swr";

import { presenceHeartbeat } from "@/api/openapi-client/misc";
import { useSession } from "@/auth";

import { useCurrentPresenceCategory } from "./presence-category";

const HEARTBEAT_INTERVAL = 20_000;

export function usePresence() {
  const session = useSession();
  const category = useCurrentPresenceCategory();

  const { data } = useSWR(
    session ? ["presence", category?.id ?? null] : null,
    ([, categoryId]) =>
      presenceHeartbeat(categoryId ? { category_id: categoryId } : {}),
    {
      refreshInterval: HEARTBEAT_INTERVAL,
      refreshWhenHidden: false,
      revalidateOnFocus: true,
      keepPreviousData: true,
      shouldRetryOnError: false,
    },
  );

  if (!session || !data) {
    return undefined;
  }

  return {
    online: data.online,
    inCategory: category ? data.in_category : undefined,
    category,
  };
}

import { Arguments, useSWRConfig } from "swr";

import { getProfileListKey } from "@/api/openapi-client/profiles";
import { ProfileReference } from "@/api/openapi-schema";
import { WithDisclosure } from "@/utils/useDisclosure";

import { handle } from "@/api/client";
import { adminAccountDelete } from "@/api/openapi-client/admin";

export type Props = {
  profile: ProfileReference;
};

export function useMemberDeletion({
  profile,
  ...props
}: WithDisclosure<Props>) {
  const { mutate } = useSWRConfig();

  const profileKey = getProfileListKey()[0];
  const keyFn = (key: Arguments) => {
    return Array.isArray(key) && key[0].startsWith(profileKey);
  };

  async function handleDelete() {
    await handle(async () => {
      await adminAccountDelete(profile.handle);

      mutate(keyFn);
      props.onClose?.();
    });
  }

  return {
    handlers: {
      handleDelete,
    },
  };
}

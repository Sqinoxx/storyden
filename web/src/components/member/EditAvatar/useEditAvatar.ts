import { useEffect, useState } from "react";
import { toast } from "sonner";

import { useAccountGetAvatar } from "@/api/openapi-client/accounts";
import { ProfileReference } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { useTranslation } from "@/lib/i18n";
import { useProfileMutations } from "@/lib/profile/mutation";
import { UseDisclosureProps } from "@/utils/useDisclosure";

export type Props = UseDisclosureProps & {
  profile: ProfileReference;
};

export function useEditAvatar(props: Props) {
  const t = useTranslation();
  const session = useSession();
  const { revalidate } = useProfileMutations(props.profile.handle);

  const { data, error } = useAccountGetAvatar(session?.handle ?? "");

  const [initialValue, setInitialValue] = useState<File | undefined>(undefined);

  useEffect(() => {
    (async () => {
      if (!data) return;

      const file = new File([data], "avatar.png");
      setInitialValue(file);
    })();
  }, [data]);

  if (!data) {
    return {
      ready: false as const,
      error,
    };
  }

  function handleSave() {
    revalidate();
    props.onClose?.();
    toast.success(t.profile.avatarUpdated, {
      description: t.profile.avatarUpdatedDescription,
    });
  }
  return {
    ready: true as const,
    initialValue,
    handleSave,
  };
}

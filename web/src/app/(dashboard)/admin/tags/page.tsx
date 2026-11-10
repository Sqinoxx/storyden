import { tagList } from "@/api/openapi-server/tags";
import { getServerSession } from "@/auth/server-session";
import { UnreadyBanner } from "@/components/site/Unready";
import { TagManagementScreen } from "@/screens/admin/TagManagementScreen";
import { canManageTags } from "@/utils/permissions";

export default async function Page() {
  try {
    const session = await getServerSession();
    if (!session || !canManageTags(session)) {
      return (
        <UnreadyBanner error="Nur mit der Berechtigung „Tags verwalten“ sichtbar." />
      );
    }

    const { data } = await tagList();
    return <TagManagementScreen initialTagList={data} />;
  } catch (error) {
    return <UnreadyBanner error={error} />;
  }
}

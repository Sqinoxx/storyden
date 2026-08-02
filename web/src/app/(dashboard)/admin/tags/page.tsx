import { tagList } from "@/api/openapi-server/tags";
import { getServerSession } from "@/auth/server-session";
import { UnreadyBanner } from "@/components/site/Unready";
import { TagManagementScreen } from "@/screens/admin/TagManagementScreen";
import { isModeratorOrAdmin } from "@/utils/permissions";

export default async function Page() {
  try {
    const session = await getServerSession();
    if (!session || !isModeratorOrAdmin(session)) {
      return (
        <UnreadyBanner error="Nur für Moderatoren und Admins sichtbar." />
      );
    }

    const { data } = await tagList();
    return <TagManagementScreen initialTagList={data} />;
  } catch (error) {
    return <UnreadyBanner error={error} />;
  }
}

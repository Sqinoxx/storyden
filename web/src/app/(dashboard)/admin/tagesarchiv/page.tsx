import { Permission } from "@/api/openapi-schema";
import { getServerSession } from "@/auth/server-session";
import { UnreadyBanner } from "@/components/site/Unready";
import { de } from "@/lib/i18n/translations/de";
import { DailyLibraryScreen } from "@/screens/admin/DailyLibraryScreen";
import { hasPermission } from "@/utils/permissions";

export default async function Page() {
  try {
    const session = await getServerSession();
    if (
      !session ||
      !hasPermission(
        session,
        Permission.ADMINISTRATOR,
        Permission.MANAGE_LIBRARY,
      )
    ) {
      return <UnreadyBanner error={de.dailyLibrary.unauthorized} />;
    }

    return <DailyLibraryScreen />;
  } catch (error) {
    return <UnreadyBanner error={error} />;
  }
}

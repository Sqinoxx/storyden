import { ListingStatus } from "@/api/openapi-schema";
import { Badge } from "@/components/ui/badge";
import { useTranslation } from "@/lib/i18n";

export function ListingStatusBadge({ status }: { status: ListingStatus }) {
  const t = useTranslation();
  if (status === ListingStatus.active) {
    return null;
  }

  return (
    <Badge
      size="sm"
      colorPalette={status === ListingStatus.sold ? "red" : "amber"}
    >
      {t.marketplace.statuses[status]}
    </Badge>
  );
}

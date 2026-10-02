import chroma from "chroma-js";

import { ReportStatus } from "@/api/openapi-schema";
import { badgeColourCSS } from "@/components/datagraph/DatagraphItemCard";
import { Badge } from "@/components/ui/badge";
import { useTranslation } from "@/lib/i18n";

const REPORT_STATUS_COLOR: Record<ReportStatus, string> = {
  [ReportStatus.submitted]: "#f59e0b",
  [ReportStatus.acknowledged]: "#3b82f6",
  [ReportStatus.resolved]: "#10b981",
};

type Props = {
  status: ReportStatus;
};

export function ReportStatusBadge({ status }: Props) {
  const t = useTranslation();
  const colour = REPORT_STATUS_COLOR[status];
  const cssVars = badgeColourCSS(colour);

  return (
    <Badge
      style={cssVars}
      backgroundColor="var(--colors-color-palette-bg)"
      borderColor="var(--colors-color-palette-bo)"
      color="var(--colors-color-palette-fg)"
      fontWeight="medium"
    >
      {t.report.status[status]}
    </Badge>
  );
}

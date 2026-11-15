"use client";

import { AccessGrantStatus, AccessPass } from "@/api/openapi-schema";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { Grid, styled } from "@/styled-system/jsx";

const selectStyle = {
  borderWidth: "thin",
  borderColor: "border.default",
  borderRadius: "sm",
  p: "2",
  bg: "bg.default",
  fontSize: { base: "md", md: "sm" },
  minW: "0",
  w: "full",
} as const;

type Props = {
  passes: AccessPass[];
  passID: string;
  onPassChange: (id: string) => void;
  query: string;
  onQueryChange: (q: string) => void;
  statuses?: AccessGrantStatus[];
  status?: AccessGrantStatus;
  onStatusChange?: (s: AccessGrantStatus) => void;
  children?: React.ReactNode;
};

export function GrantFilters({
  passes,
  passID,
  onPassChange,
  query,
  onQueryChange,
  statuses,
  status,
  onStatusChange,
  children,
}: Props) {
  const t = useTranslation();
  const m = t.access.manage;
  const withStatus = statuses !== undefined && onStatusChange !== undefined;

  return (
    <Grid
      gap="2"
      w="full"
      alignItems="center"
      gridTemplateColumns={{
        base: "repeat(2, minmax(0, 1fr))",
        md: withStatus
          ? "auto auto minmax(0, 1fr) auto auto"
          : "auto minmax(0, 1fr)",
      }}
    >
      <styled.select
        {...selectStyle}
        gridColumn={{ base: withStatus ? "auto" : "span 2", md: "auto" }}
        aria-label={m.pass}
        value={passID}
        onChange={(e) => onPassChange(e.currentTarget.value)}
      >
        <option value="">{m.allPasses}</option>
        {passes.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name}
          </option>
        ))}
      </styled.select>
      {withStatus && (
        <styled.select
          {...selectStyle}
          aria-label="Status"
          value={status}
          onChange={(e) =>
            onStatusChange(e.currentTarget.value as AccessGrantStatus)
          }
        >
          {statuses.map((s) => (
            <option key={s} value={s}>
              {t.access.status[s]}
            </option>
          ))}
        </styled.select>
      )}
      <Input
        size="sm"
        fontSize={{ base: "md", md: "sm" }}
        gridColumn={{ base: "span 2", md: "auto" }}
        placeholder={m.searchRequests}
        aria-label={m.search}
        value={query}
        onChange={(e) => onQueryChange(e.currentTarget.value)}
      />
      {children}
    </Grid>
  );
}

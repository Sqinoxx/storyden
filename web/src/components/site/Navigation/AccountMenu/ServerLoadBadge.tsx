"use client";

import Link from "next/link";

import { useAdminServerMetrics } from "@/api/openapi-client/admin";
import {
  LOAD_COLOR,
  formatDecimal,
  formatPercent,
  loadLevel,
  percentOf,
} from "@/screens/admin/ServerSettingsScreen/format";
import { Box, Flex, styled } from "@/styled-system/jsx";

const REFRESH_MS = 3000;

export function ServerLoadBadge({ active }: { active: boolean }) {
  const { data } = useAdminServerMetrics(
    { range: "live" },
    { swr: { enabled: active, refreshInterval: active ? REFRESH_MS : 0 } },
  );

  const c = data?.current;
  const cpu = c?.cpu_percent;
  const mem = c ? percentOf(c.mem_used, c.mem_total) : undefined;

  return (
    <Link href="/admin?tab=server" data-testid="server-load-badge">
      <Flex
        direction="column"
        gap="1"
        mx="1"
        my="1"
        px="2"
        py="1.5"
        borderRadius="md"
        _hover={{ bgColor: "bg.subtle" }}
      >
        <Flex justifyContent="space-between" fontSize="xs" color="fg.muted">
          <span>Server</span>
          <styled.span fontVariantNumeric="tabular-nums">
            {c ? `${formatDecimal(c.requests_per_sec)} Anfragen/s` : "…"}
          </styled.span>
        </Flex>
        <Meter label="CPU" percent={cpu} />
        <Meter label="RAM" percent={mem} />
      </Flex>
    </Link>
  );
}

function Meter({ label, percent }: { label: string; percent?: number }) {
  const value = Math.min(100, Math.max(0, percent ?? 0));

  return (
    <Flex alignItems="center" gap="2" fontSize="xs">
      <styled.span w="8" color="fg.muted" flexShrink="0">
        {label}
      </styled.span>
      <Box
        flex="1"
        h="1.5"
        borderRadius="full"
        bgColor="bg.muted"
        overflow="hidden"
      >
        <Box
          h="full"
          borderRadius="full"
          style={{
            width: `${value}%`,
            backgroundColor: LOAD_COLOR[loadLevel(value)],
          }}
        />
      </Box>
      <styled.span
        w="9"
        textAlign="right"
        color="fg.default"
        flexShrink="0"
        fontVariantNumeric="tabular-nums"
      >
        {percent === undefined ? "…" : formatPercent(percent)}
      </styled.span>
    </Flex>
  );
}

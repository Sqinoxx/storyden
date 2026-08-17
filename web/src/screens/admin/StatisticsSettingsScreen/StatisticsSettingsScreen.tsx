"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import {
  Activity,
  Clock,
  FolderTree,
  LogIn,
  MessageSquare,
  MessagesSquare,
  ShieldOff,
  UserCheck,
  Users,
} from "lucide-react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  TooltipContentProps,
  XAxis,
  YAxis,
} from "recharts";

import { useAdminStatistics } from "@/api/openapi-client/admin";
import type {
  AdminStatistics200,
  StatisticsContributor,
  StatisticsSeriesPoint,
} from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import * as Table from "@/components/ui/table";
import { ProfileRoute } from "@/components/site/Navigation/Anchors/Profile";
import { FINISHED_SEMESTER } from "@/lib/profile/academic";
import { Box, Flex, Grid, HStack, Stack, styled } from "@/styled-system/jsx";

type Granularity = "daily" | "monthly" | "yearly";

const GRANULARITY_LABEL: Record<Granularity, string> = {
  daily: "Täglich",
  monthly: "Monatlich",
  yearly: "Jährlich",
};

export function StatisticsSettingsScreen() {
  const { data, isLoading } = useAdminStatistics();
  const [granularity, setGranularity] = useState<Granularity>("daily");
  const [assetsGranularity, setAssetsGranularity] =
    useState<Granularity>("daily");

  const growthData = useMemo(
    () => buildGrowthData(data, granularity),
    [data, granularity],
  );

  const loginInsight = useMemo(() => buildLoginInsight(data), [data]);

  const assetsGrowthData = useMemo(() => {
    const series = data
      ? selectByGranularity(
          assetsGranularity,
          data.assetsDaily,
          data.assetsMonthly,
          data.assetsYearly,
        )
      : [];
    return series.map((point) => ({
      label: formatDate(point.date, assetsGranularity),
      count: point.count,
    }));
  }, [data, assetsGranularity]);

  const loginsByHourData = useMemo(
    () =>
      (data?.loginsByHour ?? []).map((point) => ({
        hour: `${point.hour}`,
        count: point.count,
      })),
    [data],
  );

  const loginsByWeekdayData = useMemo(
    () =>
      (data?.loginsByWeekday ?? []).map((point) => ({
        weekday: formatWeekday(point.weekday),
        count: point.count,
      })),
    [data],
  );

  const topCategoriesData = useMemo(
    () =>
      (data?.topCategories ?? []).map((point) => ({
        name: point.name,
        count: point.threadCount,
      })),
    [data],
  );

  const semesterData = useMemo(
    () =>
      (data?.threadsBySemester ?? []).map((point) => ({
        term: formatTerm(point.term),
        count: point.count,
      })),
    [data],
  );

  const fachsemesterData = useMemo(
    () =>
      (data?.threadsByFachsemester ?? []).map((point) => ({
        semester: formatFachsemester(point.semester),
        count: point.count,
      })),
    [data],
  );

  const fachsemesterInsight = useMemo(
    () => buildFachsemesterInsight(data?.threadsByFachsemester ?? []),
    [data],
  );

  const assetsFachsemesterData = useMemo(
    () =>
      (data?.assetsByFachsemester ?? []).map((point) => ({
        semester: formatFachsemester(point.semester),
        count: point.count,
      })),
    [data],
  );

  return (
    <Stack gap="6" width="full">
      <Box borderBottomWidth="thin" borderColor="border.subtle" pb="4">
        <styled.h2 fontSize="xl" fontWeight="bold" color="fg.default">
          Nutzungsstatistiken
        </styled.h2>
        <styled.p fontSize="sm" color="fg.muted" mt="1">
          Überblick über Wachstum und Aktivität der Community: Mitglieder,
          Themen und Beiträge im Zeitverlauf sowie nach Semester.
        </styled.p>
      </Box>

      <Grid columns={{ base: 2, md: 3, lg: 6 }} gap="4">
        <StatCard
          label="Mitglieder"
          value={data?.totals.accounts}
          loading={isLoading}
          icon={<Users size={18} color="var(--colors-green-9)" />}
        />
        <StatCard
          label="Themen"
          value={data?.totals.threads}
          loading={isLoading}
          icon={<MessageSquare size={18} color="var(--colors-blue-9)" />}
        />
        <StatCard
          label="Antworten"
          value={data?.totals.replies}
          loading={isLoading}
          icon={<MessagesSquare size={18} color="var(--colors-green-9)" />}
        />
        <StatCard
          label="Kategorien"
          value={data?.totals.categories}
          loading={isLoading}
          icon={<FolderTree size={18} color="var(--colors-amber-9)" />}
        />
        <StatCard
          label="Aktiv (7 Tage)"
          value={data?.totals.activeAccounts7d}
          loading={isLoading}
          icon={<Activity size={18} color="var(--colors-accent-9)" />}
        />
        <StatCard
          label="Aktiv (30 Tage)"
          value={data?.totals.activeAccounts30d}
          loading={isLoading}
          icon={<UserCheck size={18} color="var(--colors-accent-9)" />}
        />
        <StatCard
          label="Sitzungen aktiv"
          value={data?.totals.sessionsActive}
          loading={isLoading}
          icon={<LogIn size={18} color="var(--colors-green-9)" />}
        />
        <StatCard
          label="Sitzungen abgelaufen"
          value={data?.totals.sessionsExpired}
          loading={isLoading}
          icon={<Clock size={18} color="var(--colors-amber-9)" />}
        />
        <StatCard
          label="Sitzungen widerrufen"
          value={data?.totals.sessionsRevoked}
          loading={isLoading}
          icon={<ShieldOff size={18} color="var(--colors-red-9)" />}
        />
      </Grid>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <Flex
          justifyContent="space-between"
          alignItems="center"
          flexWrap="wrap"
          gap="3"
          mb="4"
        >
          <Box>
            <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
              Wachstum
            </styled.h3>
            <styled.p fontSize="sm" color="fg.muted">
              Neue Mitglieder, neue Themen, Logins und aktive Nutzer im
              Zeitverlauf.
            </styled.p>
          </Box>
          <HStack gap="1">
            {(Object.keys(GRANULARITY_LABEL) as Granularity[]).map((g) => (
              <Button
                key={g}
                type="button"
                size="sm"
                variant={granularity === g ? "solid" : "ghost"}
                onClick={() => setGranularity(g)}
              >
                {GRANULARITY_LABEL[g]}
              </Button>
            ))}
          </HStack>
        </Flex>

        <Box height="72" width="full">
          {isLoading ? (
            <ChartPlaceholder />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart
                data={growthData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  stroke="var(--colors-border-subtle)"
                  vertical={false}
                />
                <XAxis
                  dataKey="label"
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  minTickGap={16}
                />
                <YAxis
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  width={32}
                />
                <Tooltip content={GrowthTooltip} />
                <Legend
                  verticalAlign="top"
                  height={32}
                  iconType="circle"
                  wrapperStyle={{ fontSize: 12 }}
                />
                <Line
                  type="monotone"
                  dataKey="accounts"
                  name="Mitglieder"
                  stroke="var(--colors-green-9)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
                <Line
                  type="monotone"
                  dataKey="threads"
                  name="Themen"
                  stroke="var(--colors-blue-9)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
                <Line
                  type="monotone"
                  dataKey="logins"
                  name="Logins"
                  stroke="var(--colors-red-9)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
                <Line
                  type="monotone"
                  dataKey="activeAccounts"
                  name="Aktive Nutzer"
                  stroke="var(--colors-amber-9)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </Box>

        {loginInsight && (
          <styled.p fontSize="sm" color="fg.default" mt="4">
            Heute: {loginInsight.loginsToday} Logins, davon ca.{" "}
            <styled.span fontWeight="semibold">
              {loginInsight.returning} wiederkehrend
            </styled.span>{" "}
            ({loginInsight.newSignups} Neuregistrierungen).
          </styled.p>
        )}
        <styled.p fontSize="xs" color="fg.muted" mt="1">
          Logins nähern sich über neue Sitzungen an: da bei jeder
          Registrierung ebenfalls eine Sitzung entsteht, sind Tage mit vielen
          Neuanmeldungen entsprechend höher.
        </styled.p>
      </Box>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <Flex
          justifyContent="space-between"
          alignItems="center"
          flexWrap="wrap"
          gap="3"
          mb="4"
        >
          <Box>
            <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
              Datei-Uploads
            </styled.h3>
            <styled.p fontSize="sm" color="fg.muted">
              Neu hochgeladene Dateien im Zeitverlauf.
            </styled.p>
          </Box>
          <HStack gap="1">
            {(Object.keys(GRANULARITY_LABEL) as Granularity[]).map((g) => (
              <Button
                key={g}
                type="button"
                size="sm"
                variant={assetsGranularity === g ? "solid" : "ghost"}
                onClick={() => setAssetsGranularity(g)}
              >
                {GRANULARITY_LABEL[g]}
              </Button>
            ))}
          </HStack>
        </Flex>

        <Box height="72" width="full">
          {isLoading ? (
            <ChartPlaceholder />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <LineChart
                data={assetsGrowthData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  stroke="var(--colors-border-subtle)"
                  vertical={false}
                />
                <XAxis
                  dataKey="label"
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  minTickGap={16}
                />
                <YAxis
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  width={32}
                />
                <Tooltip content={FilesTooltip} />
                <Line
                  type="monotone"
                  dataKey="count"
                  name="Dateien"
                  stroke="var(--colors-amber-9)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
              </LineChart>
            </ResponsiveContainer>
          )}
        </Box>
      </Box>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
          Logins nach Uhrzeit
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb="4">
          Zu welchen Tageszeiten (UTC) die meisten Logins stattfinden.
        </styled.p>

        <Box height="72" width="full">
          {isLoading ? (
            <ChartPlaceholder />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={loginsByHourData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  stroke="var(--colors-border-subtle)"
                  vertical={false}
                />
                <XAxis
                  dataKey="hour"
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  interval={1}
                />
                <YAxis
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  width={32}
                />
                <Tooltip content={LoginsTooltip} />
                <Bar
                  dataKey="count"
                  name="Logins"
                  fill="var(--colors-blue-9)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Box>
      </Box>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
          Logins nach Wochentag
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb="4">
          An welchen Wochentagen die meisten Logins stattfinden.
        </styled.p>

        <Box height="72" width="full">
          {isLoading ? (
            <ChartPlaceholder />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={loginsByWeekdayData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  stroke="var(--colors-border-subtle)"
                  vertical={false}
                />
                <XAxis
                  dataKey="weekday"
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  width={32}
                />
                <Tooltip content={LoginsTooltip} />
                <Bar
                  dataKey="count"
                  name="Logins"
                  fill="var(--colors-green-9)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Box>
      </Box>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
          Neue Themen je Semester
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb="4">
          Anzahl neu erstellter Themen, gebündelt nach deutschem
          Studiensemester (Winter-/Sommersemester).
        </styled.p>

        <Box height="72" width="full">
          {isLoading ? (
            <ChartPlaceholder />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={semesterData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  stroke="var(--colors-border-subtle)"
                  vertical={false}
                />
                <XAxis
                  dataKey="term"
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  width={32}
                />
                <Tooltip content={SemesterTooltip} />
                <Bar
                  dataKey="count"
                  name="Themen"
                  fill="var(--colors-blue-9)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Box>
      </Box>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
          Themen nach Fachsemester
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb={fachsemesterInsight ? "2" : "4"}>
          Wie viele Themen Mitglieder je nach ihrem aktuellen Fachsemester
          erstellt haben. So lässt sich erkennen, welche Semester besonders
          kollegial mitwirken und welche eher zurückhaltend sind.
        </styled.p>
        {fachsemesterInsight && (
          <styled.p fontSize="sm" color="fg.default" mb="4">
            Am aktivsten:{" "}
            <styled.span fontWeight="semibold">
              {fachsemesterInsight.mostLabel}
            </styled.span>{" "}
            ({fachsemesterInsight.mostCount} Themen) · Am zurückhaltendsten:{" "}
            <styled.span fontWeight="semibold">
              {fachsemesterInsight.leastLabel}
            </styled.span>{" "}
            ({fachsemesterInsight.leastCount} Themen)
          </styled.p>
        )}

        <Box height="72" width="full">
          {isLoading ? (
            <ChartPlaceholder />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={fachsemesterData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  stroke="var(--colors-border-subtle)"
                  vertical={false}
                />
                <XAxis
                  dataKey="semester"
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  width={32}
                />
                <Tooltip content={SemesterTooltip} />
                <Bar
                  dataKey="count"
                  name="Themen"
                  fill="var(--colors-green-9)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Box>
      </Box>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
          Dateien nach Fachsemester
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb="4">
          Wie viele Dateien Mitglieder je nach ihrem aktuellen Fachsemester
          hochgeladen haben.
        </styled.p>

        <Box height="72" width="full">
          {isLoading ? (
            <ChartPlaceholder />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={assetsFachsemesterData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  stroke="var(--colors-border-subtle)"
                  vertical={false}
                />
                <XAxis
                  dataKey="semester"
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  width={32}
                />
                <Tooltip content={FilesTooltip} />
                <Bar
                  dataKey="count"
                  name="Dateien"
                  fill="var(--colors-amber-9)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Box>
      </Box>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
          Aktivste Kategorien
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb="4">
          Die Kategorien mit den meisten neuen Themen.
        </styled.p>

        <Box height="72" width="full">
          {isLoading ? (
            <ChartPlaceholder />
          ) : (
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={topCategoriesData}
                margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
              >
                <CartesianGrid
                  stroke="var(--colors-border-subtle)"
                  vertical={false}
                />
                <XAxis
                  dataKey="name"
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  stroke="var(--colors-fg-muted)"
                  fontSize={12}
                  tickLine={false}
                  axisLine={false}
                  allowDecimals={false}
                  width={32}
                />
                <Tooltip content={SemesterTooltip} />
                <Bar
                  dataKey="count"
                  name="Themen"
                  fill="var(--colors-accent-9)"
                  radius={[4, 4, 0, 0]}
                />
              </BarChart>
            </ResponsiveContainer>
          )}
        </Box>
      </Box>

      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
          Aktivste Mitglieder
        </styled.h3>
        <styled.p fontSize="sm" color="fg.muted" mb="4">
          Die Mitglieder mit den meisten erstellten Themen, mit Fachsemester
          und Datum ihres letzten Themas.
        </styled.p>

        {isLoading ? (
          <ChartPlaceholder />
        ) : (
          <ContributorsTable contributors={data?.topContributors ?? []} />
        )}
      </Box>
    </Stack>
  );
}

function buildGrowthData(
  data: AdminStatistics200 | undefined,
  granularity: Granularity,
) {
  if (!data) return [];

  const accounts = pickSeries(data, granularity, "accounts");
  const threads = pickSeries(data, granularity, "threads");
  const logins = pickSeries(data, granularity, "logins");
  const activeAccounts = pickSeries(data, granularity, "activeAccounts");

  return accounts.map((point, i) => ({
    label: formatDate(point.date, granularity),
    accounts: point.count,
    threads: threads[i]?.count ?? 0,
    logins: logins[i]?.count ?? 0,
    activeAccounts: activeAccounts[i]?.count ?? 0,
  }));
}

function selectByGranularity<T>(
  granularity: Granularity,
  daily: T[],
  monthly: T[],
  yearly: T[],
): T[] {
  switch (granularity) {
    case "daily":
      return daily;
    case "monthly":
      return monthly;
    case "yearly":
      return yearly;
  }
}

function pickSeries(
  data: AdminStatistics200,
  granularity: Granularity,
  kind: "accounts" | "threads" | "logins" | "activeAccounts",
): StatisticsSeriesPoint[] {
  switch (kind) {
    case "accounts":
      return selectByGranularity(
        granularity,
        data.accountsDaily,
        data.accountsMonthly,
        data.accountsYearly,
      );
    case "threads":
      return selectByGranularity(
        granularity,
        data.threadsDaily,
        data.threadsMonthly,
        data.threadsYearly,
      );
    case "logins":
      return selectByGranularity(
        granularity,
        data.loginsDaily,
        data.loginsMonthly,
        data.loginsYearly,
      );
    case "activeAccounts":
      return selectByGranularity(
        granularity,
        data.activeAccountsDaily,
        data.activeAccountsMonthly,
        data.activeAccountsYearly,
      );
  }
}

// buildLoginInsight nets today's new-signup sessions out of today's total
// login count, so the "raw" logins-includes-signups caveat becomes a
// concrete, readable number rather than just a disclaimer.
function buildLoginInsight(data: AdminStatistics200 | undefined) {
  if (!data) return null;

  const loginsToday = data.loginsDaily.at(-1)?.count ?? 0;
  const newSignups = data.accountsDaily.at(-1)?.count ?? 0;

  if (loginsToday === 0) return null;

  return {
    loginsToday,
    newSignups,
    returning: Math.max(0, loginsToday - newSignups),
  };
}

function formatDate(date: string, granularity: Granularity): string {
  const parsed = new Date(`${date}T00:00:00Z`);

  switch (granularity) {
    case "daily":
      return parsed.toLocaleDateString("de-DE", {
        day: "2-digit",
        month: "2-digit",
        timeZone: "UTC",
      });
    case "monthly":
      return parsed.toLocaleDateString("de-DE", {
        month: "short",
        year: "2-digit",
        timeZone: "UTC",
      });
    case "yearly":
      return parsed.toLocaleDateString("de-DE", {
        year: "numeric",
        timeZone: "UTC",
      });
  }
}

// formatTerm turns a backend term id like "2026-SS" into the conventional
// German notation, e.g. "SS 26" or "WS 25/26".
function formatTerm(term: string): string {
  const [year, season] = term.split("-");
  if (!year || !season) return term;

  const shortYear = year.slice(2);

  if (season === "WS") {
    const nextYear = String(Number(year) + 1).slice(2);
    return `WS ${shortYear}/${nextYear}`;
  }

  return `SS ${shortYear}`;
}

function formatFachsemester(semester: number): string {
  if (semester === 0) return "Unbekannt";
  if (semester === FINISHED_SEMESTER) return "Fertig";
  return `${semester}. Sem.`;
}

const WEEKDAY_LABELS = ["Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"];

// formatWeekday maps the backend's 1=Monday..7=Sunday convention to a short
// German label.
function formatWeekday(weekday: number): string {
  return WEEKDAY_LABELS[weekday - 1] ?? `${weekday}`;
}

function buildFachsemesterInsight(
  points: { semester: number; count: number }[],
) {
  const known = points.filter(
    (p) => p.semester !== 0 && p.semester !== FINISHED_SEMESTER,
  );
  if (known.length === 0) return null;

  const most = known.reduce((a, b) => (b.count > a.count ? b : a));
  const least = known.reduce((a, b) => (b.count < a.count ? b : a));

  if (most.count === 0 && least.count === 0) return null;

  return {
    mostLabel: formatFachsemester(most.semester),
    mostCount: most.count,
    leastLabel: formatFachsemester(least.semester),
    leastCount: least.count,
  };
}

function ContributorsTable({
  contributors,
}: {
  contributors: StatisticsContributor[];
}) {
  if (contributors.length === 0) {
    return (
      <styled.p fontSize="sm" color="fg.muted">
        Noch keine Themen erstellt.
      </styled.p>
    );
  }

  return (
    <styled.div w="full" overflowX="auto">
      <Table.Root size="sm" variant="dense">
        <Table.Head>
          <Table.Row>
            <Table.Header>Mitglied</Table.Header>
            <Table.Header>Fachsemester</Table.Header>
            <Table.Header>Themen</Table.Header>
            <Table.Header>Letztes Thema</Table.Header>
          </Table.Row>
        </Table.Head>

        <Table.Body>
          {contributors.map((c) => (
            <Table.Row key={c.accountId}>
              <Table.Cell>
                <Link href={ProfileRoute(c.handle)}>
                  <styled.span fontWeight="medium" color="fg.default">
                    {c.name}
                  </styled.span>
                  <styled.span color="fg.muted"> @{c.handle}</styled.span>
                </Link>
              </Table.Cell>
              <Table.Cell>{formatFachsemester(c.semester)}</Table.Cell>
              <Table.Cell>{c.threadCount}</Table.Cell>
              <Table.Cell>{formatDateTime(c.lastThreadAt)}</Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </styled.div>
  );
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function ChartPlaceholder() {
  return (
    <Flex
      height="full"
      width="full"
      alignItems="center"
      justifyContent="center"
    >
      <styled.span fontSize="sm" color="fg.muted">
        Lade Statistiken…
      </styled.span>
    </Flex>
  );
}

function GrowthTooltip({
  active,
  payload,
  label,
}: TooltipContentProps) {
  if (!active || !payload?.length) return null;

  return (
    <ChartTooltipCard label={label}>
      {payload.map((entry) => (
        <TooltipRow
          key={entry.dataKey as string}
          color={entry.color}
          name={entry.name as string}
          value={entry.value as number}
        />
      ))}
    </ChartTooltipCard>
  );
}

function SemesterTooltip({
  active,
  payload,
  label,
}: TooltipContentProps) {
  if (!active || !payload?.length) return null;

  return (
    <ChartTooltipCard label={label}>
      {payload.map((entry) => (
        <TooltipRow
          key={entry.dataKey as string}
          color={entry.color}
          name="Themen"
          value={entry.value as number}
        />
      ))}
    </ChartTooltipCard>
  );
}

function LoginsTooltip({ active, payload, label }: TooltipContentProps) {
  if (!active || !payload?.length) return null;

  return (
    <ChartTooltipCard label={label}>
      {payload.map((entry) => (
        <TooltipRow
          key={entry.dataKey as string}
          color={entry.color}
          name="Logins"
          value={entry.value as number}
        />
      ))}
    </ChartTooltipCard>
  );
}

function FilesTooltip({ active, payload, label }: TooltipContentProps) {
  if (!active || !payload?.length) return null;

  return (
    <ChartTooltipCard label={label}>
      {payload.map((entry) => (
        <TooltipRow
          key={entry.dataKey as string}
          color={entry.color}
          name="Dateien"
          value={entry.value as number}
        />
      ))}
    </ChartTooltipCard>
  );
}

function ChartTooltipCard({
  label,
  children,
}: {
  label?: string | number;
  children: React.ReactNode;
}) {
  return (
    <Box
      bgColor="bg.default"
      borderWidth="thin"
      borderColor="border.subtle"
      borderRadius="md"
      boxShadow="md"
      p="3"
      fontSize="xs"
    >
      {label && (
        <styled.div fontWeight="semibold" color="fg.default" mb="1">
          {label}
        </styled.div>
      )}
      <Stack gap="1">{children}</Stack>
    </Box>
  );
}

function TooltipRow({
  color,
  name,
  value,
}: {
  color?: string;
  name: string;
  value: number;
}) {
  return (
    <Flex alignItems="center" gap="2">
      <span
        style={{
          display: "inline-block",
          width: 8,
          height: 8,
          borderRadius: 9999,
          backgroundColor: color,
          flexShrink: 0,
        }}
      />
      <styled.span color="fg.muted">{name}:</styled.span>
      <styled.span fontWeight="semibold" color="fg.default">
        {value}
      </styled.span>
    </Flex>
  );
}

function StatCard({
  label,
  value,
  loading,
  icon,
}: {
  label: string;
  value: number | undefined;
  loading: boolean;
  icon: React.ReactNode;
}) {
  return (
    <Box
      p="4"
      borderRadius="lg"
      borderWidth="thin"
      borderColor="border.subtle"
      bgColor="bg.subtle"
    >
      <Flex alignItems="center" justifyContent="space-between">
        <styled.span
          fontSize="xs"
          fontWeight="semibold"
          color="fg.muted"
          textTransform="uppercase"
        >
          {label}
        </styled.span>
        {icon}
      </Flex>
      <styled.div fontSize="2xl" fontWeight="bold" color="fg.default" mt="2">
        {loading ? "…" : (value ?? 0)}
      </styled.div>
    </Box>
  );
}

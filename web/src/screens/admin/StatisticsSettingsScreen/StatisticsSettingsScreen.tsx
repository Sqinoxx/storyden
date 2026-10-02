"use client";

import {
  FileUp,
  Heart,
  LogIn,
  MessageSquare,
  MessagesSquare,
  UserPlus,
} from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";

import { useAdminStatistics } from "@/api/openapi-client/admin";
import type {
  AdminStatistics200,
  StatisticsContributor,
  StatisticsThreadPoint,
} from "@/api/openapi-schema";
import { ProfileRoute } from "@/components/site/Navigation/Anchors/Profile";
import { ReportsRoute } from "@/components/site/Navigation/Anchors/Reports";
import { TagsRoute } from "@/components/site/Navigation/Anchors/Tags";
import * as Table from "@/components/ui/table";
import { formatTermKeyLabel } from "@/lib/thread/semester";
import { Box, Flex, Grid, Stack, styled } from "@/styled-system/jsx";

import { COLORS, MultiLineChart, SimpleBarChart } from "./charts";
import {
  Granularity,
  buildFachsemesterInsight,
  formatDateTime,
  formatFachsemester,
  formatNumber,
  formatWeekday,
  formatWeekdayName,
  isEmpty,
  mergeSeries,
  peak,
  sumSeries,
  sumTrends,
} from "./format";
import {
  ChartCard,
  GranularityToggle,
  KpiCard,
  StatSection,
  TotalsStrip,
} from "./layout";

export function StatisticsSettingsScreen() {
  const { data, isLoading } = useAdminStatistics();

  return (
    <Stack gap="10" width="full">
      <Box borderBottomWidth="thin" borderColor="border.subtle" pb="4">
        <styled.h2 fontSize="xl" fontWeight="bold" color="fg.default">
          Nutzungsstatistiken
        </styled.h2>
        <styled.p fontSize="sm" color="fg.muted" mt="1">
          Wachstum, Engagement und Aktivität der Community auf einen Blick.
          Trends vergleichen die letzten 30 Tage mit den 30 Tagen davor.
        </styled.p>
      </Box>

      <OverviewSection data={data} loading={isLoading} />
      <GrowthSection data={data} loading={isLoading} />
      <EngagementSection data={data} loading={isLoading} />
      <ActivitySection data={data} loading={isLoading} />
      <StudySection data={data} loading={isLoading} />
      <ContentSection data={data} loading={isLoading} />
    </Stack>
  );
}

type SectionProps = {
  data: AdminStatistics200 | undefined;
  loading: boolean;
};

const ICON_SIZE = 16;

function OverviewSection({ data, loading }: SectionProps) {
  const trends = data?.trends;
  const totals = data?.totals;

  return (
    <StatSection title="Überblick" description="Letzte 30 Tage">
      <Grid columns={{ base: 1, sm: 2, xl: 3 }} gap="4">
        <KpiCard
          label="Neue Mitglieder"
          icon={<UserPlus size={ICON_SIZE} color={COLORS.accounts} />}
          color={COLORS.accounts}
          trend={trends?.accounts}
          sparkline={data?.accountsDaily.map((p) => p.count)}
          loading={loading}
        />
        <KpiCard
          label="Neue Themen"
          icon={<MessageSquare size={ICON_SIZE} color={COLORS.threads} />}
          color={COLORS.threads}
          trend={trends?.threads}
          sparkline={data?.threadsDaily.map((p) => p.count)}
          loading={loading}
        />
        <KpiCard
          label="Antworten"
          icon={<MessagesSquare size={ICON_SIZE} color={COLORS.replies} />}
          color={COLORS.replies}
          trend={trends?.replies}
          loading={loading}
        />
        <KpiCard
          label="Logins"
          icon={<LogIn size={ICON_SIZE} color={COLORS.logins} />}
          color={COLORS.logins}
          trend={trends?.logins}
          sparkline={data?.loginsDaily.map((p) => p.count)}
          loading={loading}
        />
        <KpiCard
          label="Likes & Reaktionen"
          icon={<Heart size={ICON_SIZE} color={COLORS.likes} />}
          color={COLORS.likes}
          trend={trends && sumTrends(trends.likes, trends.reacts)}
          sparkline={data && sumSeries(data.likesDaily, data.reactsDaily)}
          loading={loading}
        />
        <KpiCard
          label="Datei-Uploads"
          icon={<FileUp size={ICON_SIZE} color={COLORS.assets} />}
          color={COLORS.assets}
          trend={trends?.assets}
          sparkline={data?.assetsDaily.map((p) => p.count)}
          loading={loading}
        />
      </Grid>

      <TotalsStrip
        loading={loading}
        items={[
          { label: "Mitglieder gesamt", value: totals?.accounts },
          { label: "Themen gesamt", value: totals?.threads },
          { label: "Antworten gesamt", value: totals?.replies },
          { label: "Kategorien", value: totals?.categories },
          { label: "Tags", value: totals?.tags },
          { label: "Aktive Autoren (7 T.)", value: totals?.activeAccounts7d },
          { label: "Aktive Autoren (30 T.)", value: totals?.activeAccounts30d },
          { label: "Sitzungen aktiv", value: totals?.sessionsActive },
          { label: "Sitzungen abgelaufen", value: totals?.sessionsExpired },
          { label: "Sitzungen widerrufen", value: totals?.sessionsRevoked },
        ]}
      />
    </StatSection>
  );
}

function GrowthSection({ data, loading }: SectionProps) {
  const [granularity, setGranularity] = useState<Granularity>("daily");
  const [assetsGranularity, setAssetsGranularity] =
    useState<Granularity>("daily");

  const growth = useMemo(
    () =>
      mergeSeries(
        data,
        ["accounts", "threads", "logins", "activeAccounts"],
        granularity,
      ),
    [data, granularity],
  );

  const assets = useMemo(
    () => mergeSeries(data, ["assets"], assetsGranularity),
    [data, assetsGranularity],
  );

  const loginInsight = useMemo(() => {
    if (!data) return null;
    const loginsToday = data.loginsDaily.at(-1)?.count ?? 0;
    const newSignups = data.accountsDaily.at(-1)?.count ?? 0;
    if (loginsToday === 0) return null;
    return {
      loginsToday,
      newSignups,
      returning: Math.max(0, loginsToday - newSignups),
    };
  }, [data]);

  return (
    <StatSection title="Wachstum">
      <ChartCard
        title="Community-Entwicklung"
        description="Neue Mitglieder, neue Themen, Logins und aktive Autoren."
        actions={
          <GranularityToggle value={granularity} onChange={setGranularity} />
        }
        insight={
          loginInsight && (
            <>
              Heute: {loginInsight.loginsToday} Logins, davon ca.{" "}
              <styled.strong>
                {loginInsight.returning} wiederkehrend
              </styled.strong>{" "}
              ({loginInsight.newSignups} Neuregistrierungen).
            </>
          )
        }
        footnote="Logins werden über neue Sitzungen angenähert. Da jede Registrierung ebenfalls eine Sitzung erzeugt, fallen Tage mit vielen Neuanmeldungen höher aus."
        loading={loading}
        height="72"
      >
        <MultiLineChart
          data={growth}
          series={[
            { key: "accounts", name: "Mitglieder", color: COLORS.accounts },
            { key: "threads", name: "Themen", color: COLORS.threads },
            { key: "logins", name: "Logins", color: COLORS.logins },
            {
              key: "activeAccounts",
              name: "Aktive Autoren",
              color: COLORS.activeAccounts,
            },
          ]}
        />
      </ChartCard>

      <ChartCard
        title="Datei-Uploads"
        description="Neu hochgeladene Dateien im Zeitverlauf."
        actions={
          <GranularityToggle
            value={assetsGranularity}
            onChange={setAssetsGranularity}
          />
        }
        loading={loading}
        empty={assets.every((r) => !r.assets)}
      >
        <MultiLineChart
          data={assets}
          series={[{ key: "assets", name: "Dateien", color: COLORS.assets }]}
        />
      </ChartCard>
    </StatSection>
  );
}

function EngagementSection({ data, loading }: SectionProps) {
  const [granularity, setGranularity] = useState<Granularity>("daily");

  const engagement = useMemo(
    () => mergeSeries(data, ["likes", "reacts"], granularity),
    [data, granularity],
  );

  const emojis = data?.topEmojis ?? [];
  const totals = data?.totals;

  return (
    <StatSection
      title="Engagement"
      description="Wie stark Mitglieder auf Inhalte reagieren."
    >
      <ChartCard
        title="Likes & Reaktionen"
        description={
          totals
            ? `Insgesamt ${formatNumber(totals.likes)} Likes und ${formatNumber(totals.reacts)} Reaktionen.`
            : undefined
        }
        actions={
          <GranularityToggle value={granularity} onChange={setGranularity} />
        }
        loading={loading}
        empty={engagement.every((r) => !r.likes && !r.reacts)}
      >
        <MultiLineChart
          data={engagement}
          series={[
            { key: "likes", name: "Likes", color: COLORS.likes },
            { key: "reacts", name: "Reaktionen", color: COLORS.reacts },
          ]}
        />
      </ChartCard>

      <Grid columns={{ base: 1, lg: 2 }} gap="4">
        <ChartCard
          title="Beliebteste Reaktionen"
          description="Die am häufigsten verwendeten Emojis."
          loading={loading}
          empty={emojis.length === 0}
        >
          <SimpleBarChart
            data={emojis}
            xKey="emoji"
            name="Reaktionen"
            color={COLORS.reacts}
            horizontal
          />
        </ChartCard>

        <ChartCard
          title="Meistgelikte Themen"
          description="Themen, deren Eröffnungsbeitrag die meisten Likes erhielt."
          loading={loading}
          empty={(data?.topLikedThreads.length ?? 0) === 0}
          height="auto"
        >
          <RankedThreadList threads={data?.topLikedThreads ?? []} />
        </ChartCard>
      </Grid>
    </StatSection>
  );
}

function ActivitySection({ data, loading }: SectionProps) {
  const hours = data?.loginsByHour ?? [];
  const weekdays = data?.loginsByWeekday ?? [];

  const peakHour = peak(hours);
  const peakWeekday = peak(weekdays);

  const hourData = hours.map((p) => ({ hour: `${p.hour}`, count: p.count }));
  const weekdayData = weekdays.map((p) => ({
    weekday: formatWeekday(p.weekday),
    count: p.count,
  }));

  return (
    <StatSection
      title="Aktivitätsmuster"
      description="Wann sich Mitglieder anmelden (deutsche Zeit)."
    >
      <Grid columns={{ base: 1, lg: 2 }} gap="4">
        <ChartCard
          title="Logins nach Uhrzeit"
          insight={
            peakHour && (
              <>
                Spitzenzeit:{" "}
                <styled.strong>
                  {peakHour.hour}–{(peakHour.hour + 1) % 24} Uhr
                </styled.strong>
              </>
            )
          }
          loading={loading}
          empty={isEmpty(hours)}
        >
          <SimpleBarChart
            data={hourData}
            xKey="hour"
            name="Logins"
            color={COLORS.threads}
            highlight={peakHour && `${peakHour.hour}`}
          />
        </ChartCard>

        <ChartCard
          title="Logins nach Wochentag"
          insight={
            peakWeekday && (
              <>
                Aktivster Tag:{" "}
                <styled.strong>
                  {formatWeekdayName(peakWeekday.weekday)}
                </styled.strong>
              </>
            )
          }
          loading={loading}
          empty={isEmpty(weekdays)}
        >
          <SimpleBarChart
            data={weekdayData}
            xKey="weekday"
            name="Logins"
            color={COLORS.accounts}
            highlight={peakWeekday && formatWeekday(peakWeekday.weekday)}
          />
        </ChartCard>
      </Grid>
    </StatSection>
  );
}

function StudySection({ data, loading }: SectionProps) {
  const semesterData = (data?.threadsBySemester ?? []).map((p) => ({
    term: formatTermKeyLabel(p.term),
    count: p.count,
  }));

  const threadsByFachsemester = data?.threadsByFachsemester ?? [];
  const assetsByFachsemester = data?.assetsByFachsemester ?? [];
  const insight = buildFachsemesterInsight(threadsByFachsemester);

  return (
    <StatSection
      title="Studium"
      description="Aktivität nach Studiensemester und Fachsemester der Mitglieder."
    >
      <ChartCard
        title="Neue Themen je Semester"
        description="Neu erstellte Themen, gebündelt nach Winter-/Sommersemester."
        loading={loading}
        empty={isEmpty(data?.threadsBySemester ?? [])}
      >
        <SimpleBarChart
          data={semesterData}
          xKey="term"
          name="Themen"
          color={COLORS.threads}
        />
      </ChartCard>

      <Grid columns={{ base: 1, lg: 2 }} gap="4">
        <ChartCard
          title="Themen nach Fachsemester"
          description="Welche Semester besonders aktiv mitwirken."
          insight={
            insight && (
              <>
                Am aktivsten: <styled.strong>{insight.mostLabel}</styled.strong>{" "}
                ({insight.mostCount}) · Am zurückhaltendsten:{" "}
                <styled.strong>{insight.leastLabel}</styled.strong> (
                {insight.leastCount})
              </>
            )
          }
          loading={loading}
          empty={isEmpty(threadsByFachsemester)}
        >
          <SimpleBarChart
            data={threadsByFachsemester.map((p) => ({
              semester: formatFachsemester(p.semester),
              count: p.count,
            }))}
            xKey="semester"
            name="Themen"
            color={COLORS.accounts}
          />
        </ChartCard>

        <ChartCard
          title="Dateien nach Fachsemester"
          description="Welche Semester das meiste Material hochladen."
          loading={loading}
          empty={isEmpty(assetsByFachsemester)}
        >
          <SimpleBarChart
            data={assetsByFachsemester.map((p) => ({
              semester: formatFachsemester(p.semester),
              count: p.count,
            }))}
            xKey="semester"
            name="Dateien"
            color={COLORS.assets}
          />
        </ChartCard>
      </Grid>
    </StatSection>
  );
}

function ContentSection({ data, loading }: SectionProps) {
  const categories = (data?.topCategories ?? []).map((c) => ({
    name: c.name,
    count: c.threadCount,
  }));
  const tags = (data?.topTags ?? []).map((t) => ({
    name: t.name,
    count: t.threadCount,
  }));

  return (
    <StatSection
      title="Inhalte & Moderation"
      description="Wo diskutiert wird, wer am aktivsten ist und was gemeldet wurde."
    >
      <Grid columns={{ base: 1, lg: 2 }} gap="4">
        <ChartCard
          title="Aktivste Kategorien"
          description="Kategorien mit den meisten neuen Themen."
          loading={loading}
          empty={categories.length === 0}
          height="72"
        >
          <SimpleBarChart
            data={categories}
            xKey="name"
            name="Themen"
            color={COLORS.categories}
            horizontal
          />
        </ChartCard>

        <ChartCard
          title="Beliebteste Tags"
          description="Tags, die an den meisten Themen hängen."
          actions={
            <styled.span fontSize="sm">
              <Link href={TagsRoute}>Alle Tags</Link>
            </styled.span>
          }
          loading={loading}
          empty={tags.length === 0}
          height="72"
        >
          <SimpleBarChart
            data={tags}
            xKey="name"
            name="Themen"
            color={COLORS.tags}
            horizontal
          />
        </ChartCard>
      </Grid>

      <Grid columns={{ base: 1, lg: 3 }} gap="4">
        <ReportsCard data={data} loading={loading} />

        <Box gridColumn={{ lg: "span 2" }} minW="0">
          <ChartCard
            title="Aktivste Mitglieder"
            description="Meiste erstellte Themen, mit Fachsemester und letztem Thema."
            loading={loading}
            empty={(data?.topContributors.length ?? 0) === 0}
            height="auto"
          >
            <ContributorsTable contributors={data?.topContributors ?? []} />
          </ChartCard>
        </Box>
      </Grid>
    </StatSection>
  );
}

function ReportsCard({ data, loading }: SectionProps) {
  const totals = data?.totals;

  const rows = [
    {
      label: "Offen",
      value: totals?.reportsSubmitted,
      color: "var(--colors-red-9)",
    },
    {
      label: "In Bearbeitung",
      value: totals?.reportsAcknowledged,
      color: "var(--colors-amber-9)",
    },
    {
      label: "Erledigt",
      value: totals?.reportsResolved,
      color: "var(--colors-green-9)",
    },
  ];

  return (
    <ChartCard
      title="Meldungen"
      description={
        totals
          ? `${formatNumber(totals.reportsLast30d)} neue Meldungen in den letzten 30 Tagen.`
          : undefined
      }
      loading={loading}
      height="auto"
    >
      <Stack gap="3">
        {rows.map((row) => (
          <Flex key={row.label} alignItems="center" gap="3">
            <styled.span
              display="inline-block"
              w="2.5"
              h="2.5"
              borderRadius="full"
              flexShrink="0"
              style={{ backgroundColor: row.color }}
            />
            <styled.span fontSize="sm" color="fg.muted" flexGrow="1">
              {row.label}
            </styled.span>
            <styled.span
              fontSize="lg"
              fontWeight="semibold"
              color="fg.default"
              fontVariantNumeric="tabular-nums"
            >
              {formatNumber(row.value)}
            </styled.span>
          </Flex>
        ))}
        <styled.span fontSize="sm" mt="1">
          <Link href={ReportsRoute}>Zur Meldungsübersicht →</Link>
        </styled.span>
      </Stack>
    </ChartCard>
  );
}

function RankedThreadList({ threads }: { threads: StatisticsThreadPoint[] }) {
  return (
    <styled.ol display="flex" flexDirection="column" gap="1">
      {threads.map((t, i) => (
        <styled.li key={t.id}>
          <Link href={`/t/locate?id=${t.id}`}>
            <Flex
              alignItems="center"
              gap="3"
              px="2"
              py="2"
              borderRadius="md"
              _hover={{ bgColor: "bg.subtle" }}
            >
              <styled.span
                w="5"
                flexShrink="0"
                fontSize="sm"
                fontWeight="semibold"
                color="fg.muted"
                fontVariantNumeric="tabular-nums"
              >
                {i + 1}.
              </styled.span>
              <styled.span
                flexGrow="1"
                minW="0"
                fontSize="sm"
                color="fg.default"
                overflow="hidden"
                textOverflow="ellipsis"
                whiteSpace="nowrap"
              >
                {t.title || "Ohne Titel"}
              </styled.span>
              <Flex
                alignItems="center"
                gap="1"
                fontSize="sm"
                fontWeight="semibold"
                color="fg.default"
                fontVariantNumeric="tabular-nums"
                flexShrink="0"
              >
                <Heart size={14} color={COLORS.likes} />
                {formatNumber(t.count)}
              </Flex>
            </Flex>
          </Link>
        </styled.li>
      ))}
    </styled.ol>
  );
}

function ContributorsTable({
  contributors,
}: {
  contributors: StatisticsContributor[];
}) {
  return (
    <styled.div w="full" overflowX="auto">
      <Table.Root size="sm" variant="dense">
        <Table.Head>
          <Table.Row>
            <Table.Header>#</Table.Header>
            <Table.Header>Mitglied</Table.Header>
            <Table.Header>Fachsemester</Table.Header>
            <Table.Header textAlign="right">Themen</Table.Header>
            <Table.Header>Letztes Thema</Table.Header>
          </Table.Row>
        </Table.Head>

        <Table.Body>
          {contributors.map((c, i) => (
            <Table.Row key={c.accountId}>
              <Table.Cell color="fg.muted">{i + 1}</Table.Cell>
              <Table.Cell>
                <Link href={ProfileRoute(c.handle)}>
                  <styled.span fontWeight="medium" color="fg.default">
                    {c.name}
                  </styled.span>
                  <styled.span color="fg.muted"> @{c.handle}</styled.span>
                </Link>
              </Table.Cell>
              <Table.Cell>{formatFachsemester(c.semester)}</Table.Cell>
              <Table.Cell
                textAlign="right"
                fontWeight="semibold"
                fontVariantNumeric="tabular-nums"
              >
                {formatNumber(c.threadCount)}
              </Table.Cell>
              <Table.Cell color="fg.muted">
                {formatDateTime(c.lastThreadAt)}
              </Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </styled.div>
  );
}

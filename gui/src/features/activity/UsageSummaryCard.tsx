import { Link as RouterLink } from "react-router-dom";
import { Alert, Badge, Box, Heading, Link, SimpleGrid, Spinner, Stat, Table, Text } from "@chakra-ui/react";
import { formatCost, formatTokens } from "../agents/components/runFormat";
import { type AgentUsageRow, USAGE_AGENT_LIMIT, type UsageSummary } from "./useActivity";
import { hasLimits, limitPressure, sortByActivity, usageTotals, withLimit } from "./usageFormat";

function PressureBadge({ row }: { row: AgentUsageRow }) {
  if (row.error) return <Badge colorPalette="red" title={row.error}>error</Badge>;
  if (!row.usage) return null;
  if (!hasLimits(row.usage)) return <Badge colorPalette="gray">no limits</Badge>;
  const p = limitPressure(row.usage);
  if (p >= 1) return <Badge colorPalette="red">at limit</Badge>;
  if (p >= 0.8) return <Badge colorPalette="orange">{Math.round(p * 100)}% of a limit</Badge>;
  return <Badge colorPalette="green">ok</Badge>;
}

function UsageRow({ row }: { row: AgentUsageRow }) {
  const u = row.usage;
  const l = u?.limits ?? {};
  const day = u?.lastDay;
  return (
    <Table.Row>
      <Table.Cell maxW="200px" overflow="hidden" textOverflow="ellipsis">
        <Link asChild color="teal.600" fontWeight="medium">
          <RouterLink to={`/agents?name=${encodeURIComponent(row.name)}`}>{row.name}</RouterLink>
        </Link>
      </Table.Cell>
      <Table.Cell textAlign="end">{u ? withLimit(String(u.lastHour.runs), l.max_runs_per_hour ? String(l.max_runs_per_hour) : null) : "–"}</Table.Cell>
      <Table.Cell textAlign="end">{day ? withLimit(String(day.runs), l.max_runs_per_day ? String(l.max_runs_per_day) : null) : "–"}</Table.Cell>
      <Table.Cell textAlign="end">
        {day ? withLimit(formatTokens(day.inputTokens + day.outputTokens), l.max_tokens_per_day ? formatTokens(l.max_tokens_per_day) : null) : "–"}
      </Table.Cell>
      <Table.Cell textAlign="end">
        {!u ? "–" : u.costKnown ? withLimit(formatCost(u.lastDay.costUsd), l.max_cost_per_day ? `$${l.max_cost_per_day.toFixed(2)}` : null) : (
          <Text as="span" color="gray.400">no prices</Text>
        )}
      </Table.Cell>
      <Table.Cell>
        <PressureBadge row={row} />
      </Table.Cell>
    </Table.Row>
  );
}

/** Usage of every agent over the last 24 hours, against its limits. */
export function UsageSummaryCard({ summary, loading, error }: { summary: UsageSummary | null; loading: boolean; error: string | null }) {
  const rows = summary ? sortByActivity(summary.rows) : [];
  const totals = usageTotals(rows);
  return (
    <Box bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 5 }} mb={6}>
      <Heading size="md" fontWeight="bold" mb={1}>
        Usage, last 24 hours
      </Heading>
      <Text fontSize="sm" color="gray.500" mb={4}>
        Per agent, against its limits. Rolling windows.
      </Text>
      {error && (
        <Alert.Root status="error" borderRadius="md">
          <Alert.Indicator />
          <Alert.Content>
            <Alert.Description>{error}</Alert.Description>
          </Alert.Content>
        </Alert.Root>
      )}
      {loading && !summary && <Spinner />}
      {summary && rows.length === 0 && (
        <Text fontSize="sm" color="gray.500">
          No agents yet.{" "}
          <Link asChild color="teal.600">
            <RouterLink to="/agents/new">Create one</RouterLink>
          </Link>
          .
        </Text>
      )}
      {rows.length > 0 && (
        <>
          <SimpleGrid columns={{ base: 2, lg: 4 }} gap={3} mb={4}>
            <Stat.Root borderWidth="1px" borderRadius="md" p={3}>
              <Stat.Label>Runs</Stat.Label>
              <Stat.ValueText fontSize="xl">{totals.runs}</Stat.ValueText>
            </Stat.Root>
            <Stat.Root borderWidth="1px" borderRadius="md" p={3}>
              <Stat.Label>Tokens</Stat.Label>
              <Stat.ValueText fontSize="xl">{formatTokens(totals.tokens)}</Stat.ValueText>
            </Stat.Root>
            <Stat.Root borderWidth="1px" borderRadius="md" p={3}>
              <Stat.Label>Cost (priced agents)</Stat.Label>
              <Stat.ValueText fontSize="xl">{totals.cost ? formatCost(totals.cost) : "$0"}</Stat.ValueText>
            </Stat.Root>
            <Stat.Root borderWidth="1px" borderRadius="md" p={3}>
              <Stat.Label>Agents at a limit</Stat.Label>
              <Stat.ValueText fontSize="xl" color={totals.atLimit > 0 ? "red.500" : undefined}>
                {totals.atLimit}
              </Stat.ValueText>
            </Stat.Root>
          </SimpleGrid>
          <Table.ScrollArea whiteSpace="nowrap" borderWidth="1px" borderRadius="md">
            <Table.Root size="sm" css={{ "& th, & td": { px: 3 } }}>
              <Table.Header>
                <Table.Row>
                  <Table.ColumnHeader>Agent</Table.ColumnHeader>
                  <Table.ColumnHeader textAlign="end">Runs, 1h</Table.ColumnHeader>
                  <Table.ColumnHeader textAlign="end">Runs, 24h</Table.ColumnHeader>
                  <Table.ColumnHeader textAlign="end">Tokens, 24h</Table.ColumnHeader>
                  <Table.ColumnHeader textAlign="end">Cost, 24h</Table.ColumnHeader>
                  <Table.ColumnHeader>Limits</Table.ColumnHeader>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {rows.map((r) => (
                  <UsageRow key={r.name} row={r} />
                ))}
              </Table.Body>
            </Table.Root>
          </Table.ScrollArea>
          {summary && summary.skipped > 0 && (
            <Text fontSize="sm" color="gray.500" mt={2}>
              Showing the first {USAGE_AGENT_LIMIT} agents; {summary.skipped} more are not summarized here. Open an agent to see its usage.
            </Text>
          )}
        </>
      )}
    </Box>
  );
}

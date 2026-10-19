import { Link as RouterLink } from "react-router-dom";
import {
  Alert,
  AlertIcon,
  Badge,
  Box,
  Heading,
  Link,
  SimpleGrid,
  Spinner,
  Stat,
  StatLabel,
  StatNumber,
  Table,
  TableContainer,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
} from "@chakra-ui/react";
import { formatCost, formatTokens } from "../agents/components/runFormat";
import { type AgentUsageRow, USAGE_AGENT_LIMIT, type UsageSummary } from "./useActivity";
import { hasLimits, limitPressure, sortByActivity, usageTotals, withLimit } from "./usageFormat";

function PressureBadge({ row }: { row: AgentUsageRow }) {
  if (row.error) return <Badge colorScheme="red" title={row.error}>error</Badge>;
  if (!row.usage) return null;
  if (!hasLimits(row.usage)) return <Badge colorScheme="gray">no limits</Badge>;
  const p = limitPressure(row.usage);
  if (p >= 1) return <Badge colorScheme="red">at limit</Badge>;
  if (p >= 0.8) return <Badge colorScheme="orange">{Math.round(p * 100)}% of a limit</Badge>;
  return <Badge colorScheme="green">ok</Badge>;
}

function UsageRow({ row }: { row: AgentUsageRow }) {
  const u = row.usage;
  const l = u?.limits ?? {};
  const day = u?.lastDay;
  return (
    <Tr>
      <Td maxW="200px" overflow="hidden" textOverflow="ellipsis">
        <Link as={RouterLink} to={`/agents?name=${encodeURIComponent(row.name)}`} color="teal.600" fontWeight="medium">
          {row.name}
        </Link>
      </Td>
      <Td isNumeric>{u ? withLimit(String(u.lastHour.runs), l.max_runs_per_hour ? String(l.max_runs_per_hour) : null) : "–"}</Td>
      <Td isNumeric>{day ? withLimit(String(day.runs), l.max_runs_per_day ? String(l.max_runs_per_day) : null) : "–"}</Td>
      <Td isNumeric>
        {day ? withLimit(formatTokens(day.inputTokens + day.outputTokens), l.max_tokens_per_day ? formatTokens(l.max_tokens_per_day) : null) : "–"}
      </Td>
      <Td isNumeric>
        {!u ? "–" : u.costKnown ? withLimit(formatCost(u.lastDay.costUsd), l.max_cost_per_day ? `$${l.max_cost_per_day.toFixed(2)}` : null) : (
          <Text as="span" color="gray.400">no prices</Text>
        )}
      </Td>
      <Td>
        <PressureBadge row={row} />
      </Td>
    </Tr>
  );
}

/** Usage of every agent over the last 24 hours, against its limits. */
export function UsageSummaryCard({ summary, loading, error }: { summary: UsageSummary | null; loading: boolean; error: string | null }) {
  const rows = summary ? sortByActivity(summary.rows) : [];
  const totals = usageTotals(rows);
  return (
    <Box bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 5 }} mb={6}>
      <Heading size="sm" mb={1}>
        Usage, last 24 hours
      </Heading>
      <Text fontSize="sm" color="gray.500" mb={4}>
        Per agent, against its limits. Rolling windows.
      </Text>
      {error && (
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          {error}
        </Alert>
      )}
      {loading && !summary && <Spinner />}
      {summary && rows.length === 0 && (
        <Text fontSize="sm" color="gray.500">
          No agents yet.{" "}
          <Link as={RouterLink} to="/agents/new" color="teal.600">
            Create one
          </Link>
          .
        </Text>
      )}
      {rows.length > 0 && (
        <>
          <SimpleGrid columns={{ base: 2, lg: 4 }} spacing={3} mb={4}>
            <Stat borderWidth="1px" borderRadius="md" p={3}>
              <StatLabel>Runs</StatLabel>
              <StatNumber fontSize="xl">{totals.runs}</StatNumber>
            </Stat>
            <Stat borderWidth="1px" borderRadius="md" p={3}>
              <StatLabel>Tokens</StatLabel>
              <StatNumber fontSize="xl">{formatTokens(totals.tokens)}</StatNumber>
            </Stat>
            <Stat borderWidth="1px" borderRadius="md" p={3}>
              <StatLabel>Cost (priced agents)</StatLabel>
              <StatNumber fontSize="xl">{totals.cost ? formatCost(totals.cost) : "$0"}</StatNumber>
            </Stat>
            <Stat borderWidth="1px" borderRadius="md" p={3}>
              <StatLabel>Agents at a limit</StatLabel>
              <StatNumber fontSize="xl" color={totals.atLimit > 0 ? "red.500" : undefined}>
                {totals.atLimit}
              </StatNumber>
            </Stat>
          </SimpleGrid>
          <TableContainer borderWidth="1px" borderRadius="md">
            <Table size="sm" sx={{ "th, td": { px: 3 } }}>
              <Thead>
                <Tr>
                  <Th>Agent</Th>
                  <Th isNumeric>Runs, 1h</Th>
                  <Th isNumeric>Runs, 24h</Th>
                  <Th isNumeric>Tokens, 24h</Th>
                  <Th isNumeric>Cost, 24h</Th>
                  <Th>Limits</Th>
                </Tr>
              </Thead>
              <Tbody>
                {rows.map((r) => (
                  <UsageRow key={r.name} row={r} />
                ))}
              </Tbody>
            </Table>
          </TableContainer>
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

import { useState } from "react";
import {
  Alert,
  AlertIcon,
  Badge,
  Box,
  Button,
  HStack,
  IconButton,
  Progress,
  Select,
  SimpleGrid,
  Spinner,
  Stat,
  StatHelpText,
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
  useColorModeValue,
} from "@chakra-ui/react";
import { FiRefreshCw } from "react-icons/fi";
import { RunStatus } from "../../../shared/types";
import { useAgentRuns } from "../hook/useAgentRuns";
import { RunDetailDrawer } from "./RunDetailDrawer";
import {
  formatCost,
  formatDuration,
  formatShortTime,
  formatTime,
  formatTokens,
  statusColor,
  triggerSummary,
} from "./runFormat";

function UsageStat({
  label,
  value,
  used,
  limit,
  format,
}: {
  label: string;
  value: string;
  used: number;
  limit?: number;
  format: (n: number) => string;
}) {
  const pct = limit ? Math.min(100, (used / limit) * 100) : 0;
  return (
    <Stat borderWidth="1px" borderRadius="md" p={3}>
      <StatLabel>{label}</StatLabel>
      <StatNumber fontSize="xl">{value}</StatNumber>
      <StatHelpText mb={limit ? 1 : 0}>{limit ? `limit ${format(limit)}` : "no limit"}</StatHelpText>
      {limit ? (
        <Progress size="xs" value={pct} colorScheme={pct >= 100 ? "red" : pct >= 80 ? "orange" : "green"} />
      ) : null}
    </Stat>
  );
}

/** Run history of an agent: usage against limits, then every run, newest first. */
export function RunsPanel({ agentName }: { agentName: string }) {
  const [status, setStatus] = useState<RunStatus | "">("");
  const [page, setPage] = useState(1);
  const [openRun, setOpenRun] = useState<number | null>(null);
  const { runs, pagination, usage, loading, error, refresh } = useAgentRuns(agentName, status, page);
  const hover = useColorModeValue("gray.50", "gray.700");

  const limits = usage?.limits ?? {};
  const day = usage?.lastDay;
  const hour = usage?.lastHour;
  const count = (n: number) => String(n);

  return (
    <Box>
      {usage && day && hour && (
        <SimpleGrid columns={{ base: 1, sm: 2, lg: 4 }} spacing={3} mb={5}>
          <UsageStat
            label="Runs, last hour"
            value={count(hour.runs)}
            used={hour.runs}
            limit={limits.max_runs_per_hour}
            format={count}
          />
          <UsageStat
            label="Runs, last 24h"
            value={count(day.runs)}
            used={day.runs}
            limit={limits.max_runs_per_day}
            format={count}
          />
          <UsageStat
            label="Tokens, last 24h"
            value={formatTokens(day.inputTokens + day.outputTokens)}
            used={day.inputTokens + day.outputTokens}
            limit={limits.max_tokens_per_day}
            format={formatTokens}
          />
          <UsageStat
            label="Cost, last 24h"
            value={usage.costKnown ? (day.costUsd ? formatCost(day.costUsd) : "$0") : "no prices set"}
            used={day.costUsd}
            limit={limits.max_cost_per_day}
            format={(n) => `$${n.toFixed(2)}`}
          />
        </SimpleGrid>
      )}
      {limits.max_tokens_per_run ? (
        <Text fontSize="sm" color="gray.500" mb={3}>
          Each run stops after {formatTokens(limits.max_tokens_per_run)} tokens.
        </Text>
      ) : null}

      <HStack mb={3} spacing={3}>
        <Select
          size="sm"
          maxW="180px"
          value={status}
          onChange={(e) => {
            setStatus(e.target.value as RunStatus | "");
            setPage(1);
          }}
        >
          <option value="">All runs</option>
          <option value="succeeded">Succeeded</option>
          <option value="failed">Failed</option>
          <option value="limited">Limited</option>
          <option value="running">Running</option>
        </Select>
        <IconButton aria-label="Refresh" size="sm" icon={<FiRefreshCw />} onClick={refresh} isLoading={loading} />
        {pagination && (
          <Text fontSize="sm" color="gray.500">
            {pagination.total} run{pagination.total === 1 ? "" : "s"}
          </Text>
        )}
      </HStack>

      {error && (
        <Alert status="error" mb={3}>
          <AlertIcon />
          {error}
        </Alert>
      )}
      {loading && runs.length === 0 && <Spinner />}
      {!loading && !error && runs.length === 0 && (
        <Text color="gray.500" fontSize="sm">
          No runs yet. Runs appear here as soon as the agent is triggered.
        </Text>
      )}

      {runs.length > 0 && (
        <TableContainer borderWidth="1px" borderRadius="md">
          <Table size="sm" sx={{ "th, td": { px: 3 } }}>
            <Thead>
              <Tr>
                <Th>Started</Th>
                <Th>Status</Th>
                <Th>Trigger</Th>
                <Th isNumeric title="Model calls / function calls">Turns</Th>
                <Th isNumeric>Tokens</Th>
                <Th isNumeric>Cost</Th>
                <Th isNumeric>Duration</Th>
              </Tr>
            </Thead>
            <Tbody>
              {runs.map((r) => (
                <Tr key={r.id} cursor="pointer" _hover={{ bg: hover }} onClick={() => setOpenRun(r.id)}>
                  <Td whiteSpace="nowrap" title={formatTime(r.startedAt)}>
                    {formatShortTime(r.startedAt)}
                  </Td>
                  <Td>
                    <Badge colorScheme={statusColor[r.status]} title={r.error || undefined}>
                      {r.status}
                    </Badge>
                  </Td>
                  <Td maxW="180px" overflow="hidden" textOverflow="ellipsis" title={triggerSummary(r.triggerType, r.trigger)}>
                    {triggerSummary(r.triggerType, r.trigger)}
                  </Td>
                  <Td isNumeric whiteSpace="nowrap">
                    {r.turns} / {r.toolCalls}
                  </Td>
                  <Td isNumeric>{formatTokens(r.inputTokens + r.outputTokens)}</Td>
                  <Td isNumeric>{formatCost(r.costUsd)}</Td>
                  <Td isNumeric>{formatDuration(r.durationMs)}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </TableContainer>
      )}

      {pagination && pagination.pageCount > 1 && (
        <HStack mt={3} justify="flex-end">
          <Button size="sm" onClick={() => setPage((p) => p - 1)} isDisabled={page <= 1}>
            Newer
          </Button>
          <Text fontSize="sm">
            {page} / {pagination.pageCount}
          </Text>
          <Button size="sm" onClick={() => setPage((p) => p + 1)} isDisabled={page >= pagination.pageCount}>
            Older
          </Button>
        </HStack>
      )}

      <RunDetailDrawer agentName={agentName} runId={openRun} onClose={() => setOpenRun(null)} />
    </Box>
  );
}

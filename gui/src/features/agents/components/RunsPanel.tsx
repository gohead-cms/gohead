import { useState } from "react";
import { Box, Button, HStack, IconButton, NativeSelect, Progress, SimpleGrid, Spinner, Stat, Table, Text } from "@chakra-ui/react";
import { useColorModeValue } from "@/components/ui/color-mode";
import { FiRefreshCw } from "react-icons/fi";
import { type RunStatus } from "../../../shared/types";
import { useAgentRuns } from "../hook/useAgentRuns";
import { RunStatusBadge } from "./RunBadges";
import { RunDetailDrawer } from "./RunDetailDrawer";
import { Notice } from "./EditorParts";
import {
  formatCost,
  formatDuration,
  formatShortTime,
  formatTime,
  formatTokens,
  RUN_STATUS_OPTIONS,
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
    <Stat.Root borderWidth="1px" borderRadius="md" p={3} gap={0}>
      <Stat.Label>{label}</Stat.Label>
      <Stat.ValueText fontSize="xl">{value}</Stat.ValueText>
      <Stat.HelpText mb={limit ? 1 : 0}>{limit ? `limit ${format(limit)}` : "no limit"}</Stat.HelpText>
      {limit ? (
        <Progress.Root size="xs" value={pct} colorPalette={pct >= 100 ? "red" : pct >= 80 ? "orange" : "green"}>
          <Progress.Track>
            <Progress.Range />
          </Progress.Track>
        </Progress.Root>
      ) : null}
    </Stat.Root>
  );
}

/** Run history of an agent: usage against limits, then every run, newest first. */
export function RunsPanel({ agentName, initialRun }: { agentName: string; initialRun?: number }) {
  const [status, setStatus] = useState<RunStatus | "">("");
  const [page, setPage] = useState(1);
  const [openRun, setOpenRun] = useState<number | null>(initialRun ?? null);
  const { runs, pagination, usage, loading, error, refresh } = useAgentRuns(agentName, status, page);
  const hover = useColorModeValue("gray.50", "gray.700");

  const limits = usage?.limits ?? {};
  const day = usage?.lastDay;
  const hour = usage?.lastHour;
  const count = (n: number) => String(n);

  return (
    <Box>
      {usage && day && hour && (
        <SimpleGrid columns={{ base: 1, sm: 2, lg: 4 }} gap={3} mb={5}>
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

      <HStack mb={3} gap={3}>
        <NativeSelect.Root size="xs" maxW="180px">
          <NativeSelect.Field
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as RunStatus | "");
              setPage(1);
            }}
          >
            <option value="">All runs</option>
            {RUN_STATUS_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </NativeSelect.Field>
          <NativeSelect.Indicator />
        </NativeSelect.Root>
        <IconButton aria-label="Refresh" size="xs" variant="subtle" onClick={refresh} loading={loading}>
          <FiRefreshCw />
        </IconButton>
        {pagination && (
          <Text fontSize="sm" color="gray.500">
            {pagination.total} run{pagination.total === 1 ? "" : "s"}
          </Text>
        )}
      </HStack>

      {error && (
        <Notice status="error" mb={3}>
          {error}
        </Notice>
      )}
      {loading && runs.length === 0 && <Spinner />}
      {!loading && !error && runs.length === 0 && (
        <Text color="gray.500" fontSize="sm">
          No runs yet. Runs appear here as soon as the agent is triggered.
        </Text>
      )}

      {runs.length > 0 && (
        <Table.ScrollArea borderWidth="1px" borderRadius="md">
          <Table.Root size="sm" css={{ "& th, & td": { px: 3 } }}>
            <Table.Header>
              <Table.Row>
                <Table.ColumnHeader>Started</Table.ColumnHeader>
                <Table.ColumnHeader>Status</Table.ColumnHeader>
                <Table.ColumnHeader>Trigger</Table.ColumnHeader>
                <Table.ColumnHeader textAlign="end" title="Model calls / function calls">
                  Turns
                </Table.ColumnHeader>
                <Table.ColumnHeader textAlign="end">Tokens</Table.ColumnHeader>
                <Table.ColumnHeader textAlign="end">Cost</Table.ColumnHeader>
                <Table.ColumnHeader textAlign="end">Duration</Table.ColumnHeader>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {runs.map((r) => (
                <Table.Row key={r.id} cursor="pointer" _hover={{ bg: hover }} onClick={() => setOpenRun(r.id)}>
                  <Table.Cell whiteSpace="nowrap" title={formatTime(r.startedAt)}>
                    {formatShortTime(r.startedAt)}
                  </Table.Cell>
                  <Table.Cell>
                    <RunStatusBadge run={r} />
                  </Table.Cell>
                  <Table.Cell maxW="180px" overflow="hidden" textOverflow="ellipsis" title={triggerSummary(r.triggerType, r.trigger)}>
                    {triggerSummary(r.triggerType, r.trigger)}
                  </Table.Cell>
                  <Table.Cell textAlign="end" whiteSpace="nowrap">
                    {r.turns} / {r.toolCalls}
                  </Table.Cell>
                  <Table.Cell textAlign="end">{formatTokens(r.inputTokens + r.outputTokens)}</Table.Cell>
                  <Table.Cell textAlign="end">{formatCost(r.costUsd)}</Table.Cell>
                  <Table.Cell textAlign="end">{formatDuration(r.durationMs)}</Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table.Root>
        </Table.ScrollArea>
      )}

      {pagination && pagination.pageCount > 1 && (
        <HStack mt={3} justify="flex-end">
          <Button size="xs" variant="subtle" onClick={() => setPage((p) => p - 1)} disabled={page <= 1}>
            Newer
          </Button>
          <Text fontSize="sm">
            {page} / {pagination.pageCount}
          </Text>
          <Button size="xs" variant="subtle" onClick={() => setPage((p) => p + 1)} disabled={page >= pagination.pageCount}>
            Older
          </Button>
        </HStack>
      )}

      <RunDetailDrawer
        agentName={agentName}
        runId={openRun}
        onClose={() => {
          setOpenRun(null);
          // The run may have ended while it was open.
          refresh();
        }}
      />
    </Box>
  );
}

import { useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import {
  Alert,
  AlertIcon,
  Badge,
  Box,
  Button,
  HStack,
  IconButton,
  Select,
  Spinner,
  Table,
  TableContainer,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
} from "@chakra-ui/react";
import { FiActivity, FiRefreshCw } from "react-icons/fi";
import { type AgentRun, type RunStatus } from "../../shared/types";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { RunDetailDrawer } from "../agents/components/RunDetailDrawer";
import {
  formatCost,
  formatDuration,
  formatShortTime,
  formatTime,
  formatTokens,
  statusColor,
  triggerSummary,
} from "../agents/components/runFormat";
import { useAllRuns, useUsageSummary } from "./useActivity";
import { UsageSummaryCard } from "./UsageSummaryCard";

/** Runs of every agent, newest first, and each agent's usage over the last 24 hours. */
export function ActivityPage() {
  const [status, setStatus] = useState<RunStatus | "">("");
  const [page, setPage] = useState(1);
  const [open, setOpen] = useState<{ agent: string; id: number } | null>(null);
  const { runs, pagination, loading, error, refetch } = useAllRuns(status, page);
  const usage = useUsageSummary();

  // Runs keep the name the agent had when they ran; use the current name
  // (by ID) so the run can be opened after a rename.
  const namesById = new Map((usage.summary?.rows ?? []).filter((r) => r.id).map((r) => [r.id, r.name]));
  const agentOf = (r: AgentRun) => (r.agentId !== undefined ? namesById.get(r.agentId) : undefined) ?? r.agentName ?? "";

  const refresh = () => {
    refetch();
    usage.refetch();
  };

  return (
    <Page>
      <PageHeader
        title="Activity"
        description="What every agent did: runs, tokens and cost."
        actions={
          <IconButton aria-label="Refresh" icon={<FiRefreshCw />} size="sm" onClick={refresh} isLoading={loading || usage.loading} />
        }
      />

      <UsageSummaryCard summary={usage.summary} loading={usage.loading} error={usage.error} />

      <Box bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 5 }}>
        <HStack mb={4} spacing={3} wrap="wrap">
          <Text fontWeight="semibold" mr="auto">
            Runs
          </Text>
          {pagination && (
            <Text fontSize="sm" color="gray.500">
              {pagination.total} run{pagination.total === 1 ? "" : "s"}
            </Text>
          )}
          <Select
            size="sm"
            maxW="180px"
            aria-label="Filter by status"
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
        </HStack>

        {error && (
          <Alert status="error" mb={3} borderRadius="md">
            <AlertIcon />
            {error}
          </Alert>
        )}
        {loading && runs.length === 0 && <Spinner />}
        {!loading && !error && runs.length === 0 &&
          (status ? (
            <Text fontSize="sm" color="gray.500">
              No {status} runs.{" "}
              <Button variant="link" size="sm" onClick={() => {
                  setStatus("");
                  setPage(1);
                }}>
                Show all runs
              </Button>
            </Text>
          ) : (
            <EmptyState
              icon={FiActivity}
              title="No runs yet"
              action={
                <Button as={RouterLink} to="/agents" size="sm" colorScheme="teal">
                  Go to agents
                </Button>
              }
            >
              Runs appear here when an agent is triggered by an event, a webhook or a schedule, or run by hand.
            </EmptyState>
          ))}

        {runs.length > 0 && (
          <TableContainer borderWidth="1px" borderRadius="md">
            <Table size="sm" sx={{ "th, td": { px: 3 } }}>
              <Thead>
                <Tr>
                  <Th>Started</Th>
                  <Th>Agent</Th>
                  <Th>Status</Th>
                  <Th>Trigger</Th>
                  <Th isNumeric title="Model calls / function calls">
                    Turns / calls
                  </Th>
                  <Th isNumeric>Tokens</Th>
                  <Th isNumeric>Cost</Th>
                  <Th isNumeric>Duration</Th>
                </Tr>
              </Thead>
              <Tbody>
                {runs.map((r) => {
                  const agent = agentOf(r);
                  const trigger = triggerSummary(r.triggerType, r.trigger);
                  return (
                    <Tr
                      key={r.id}
                      cursor="pointer"
                      _hover={{ bg: "gray.50" }}
                      tabIndex={0}
                      aria-label={`Run ${r.id} of ${agent}`}
                      onClick={() => setOpen({ agent, id: r.id })}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") setOpen({ agent, id: r.id });
                      }}
                    >
                      <Td whiteSpace="nowrap" title={formatTime(r.startedAt)}>
                        {formatShortTime(r.startedAt)}
                      </Td>
                      <Td maxW="160px" overflow="hidden" textOverflow="ellipsis" fontWeight="medium" title={agent}>
                        {agent}
                      </Td>
                      <Td>
                        <Badge colorScheme={statusColor[r.status]} title={r.error || undefined}>
                          {r.status}
                        </Badge>
                      </Td>
                      <Td maxW="180px" overflow="hidden" textOverflow="ellipsis" title={trigger}>
                        {trigger}
                      </Td>
                      <Td isNumeric whiteSpace="nowrap">
                        {r.turns} / {r.toolCalls}
                      </Td>
                      <Td isNumeric>{formatTokens(r.inputTokens + r.outputTokens)}</Td>
                      <Td isNumeric>{formatCost(r.costUsd)}</Td>
                      <Td isNumeric>{formatDuration(r.durationMs)}</Td>
                    </Tr>
                  );
                })}
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
      </Box>

      <RunDetailDrawer agentName={open?.agent ?? ""} runId={open?.id ?? null} onClose={() => setOpen(null)} />
    </Page>
  );
}

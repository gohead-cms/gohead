import { useState } from "react";
import { Link as RouterLink } from "react-router-dom";
import { Alert, Box, Button, HStack, IconButton, NativeSelect, Spinner, Table, Tabs, Text } from "@chakra-ui/react";
import { FiActivity, FiRefreshCw } from "react-icons/fi";
import { type AgentRun, type RunStatus } from "../../shared/types";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { RunStatusBadge } from "../agents/components/RunBadges";
import { RunDetailDrawer } from "../agents/components/RunDetailDrawer";
import {
  formatCost,
  formatDuration,
  formatShortTime,
  formatTime,
  formatTokens,
  RUN_STATUS_OPTIONS,
  triggerSummary,
} from "../agents/components/runFormat";
import { ContentChanges } from "../history/ContentChanges";
import { useAllRuns, useUsageSummary } from "./useActivity";
import { UsageSummaryCard } from "./UsageSummaryCard";

/**
 * Runs of every agent, newest first, and each agent's usage over the last
 * 24 hours; and the content changes of every entry, by anyone.
 */
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
        description="What every agent did, and every change to content."
        actions={
          <IconButton aria-label="Refresh" size="xs" variant="subtle" onClick={refresh} loading={loading || usage.loading}>
            <FiRefreshCw />
          </IconButton>
        }
      />

      <Tabs.Root defaultValue="runs" lazyMount unmountOnExit colorPalette="purple">
        <Tabs.List mb={4}>
          <Tabs.Trigger value="runs" _selected={{ color: "purple.600" }}>
            Agent runs
          </Tabs.Trigger>
          <Tabs.Trigger value="changes" _selected={{ color: "purple.600" }}>
            Content changes
          </Tabs.Trigger>
        </Tabs.List>
        <Tabs.Content value="runs" px={0} py={4}>
          <UsageSummaryCard summary={usage.summary} loading={usage.loading} error={usage.error} />

          <Box bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 5 }}>
            <HStack mb={4} gap={3} wrap="wrap">
              <Text fontWeight="semibold" mr="auto">
                Runs
              </Text>
              {pagination && (
                <Text fontSize="sm" color="gray.500">
                  {pagination.total} run{pagination.total === 1 ? "" : "s"}
                </Text>
              )}
              <NativeSelect.Root size="xs" maxW="180px">
                <NativeSelect.Field
                  aria-label="Filter by status"
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
            </HStack>

            {error && (
              <Alert.Root role="alert" status="error" mb={3} borderRadius="md">
                <Alert.Indicator />
                <Alert.Content>
                  <Alert.Description>{error}</Alert.Description>
                </Alert.Content>
              </Alert.Root>
            )}
            {loading && runs.length === 0 && <Spinner />}
            {!loading && !error && runs.length === 0 &&
              (status ? (
                <Text fontSize="sm" color="gray.500">
                  No {status} runs.{" "}
                  <Button
                    variant="plain"
                    size="xs"
                    color="gray.500"
                    px={0}
                    h="auto"
                    _hover={{ textDecoration: "underline" }}
                    onClick={() => {
                      setStatus("");
                      setPage(1);
                    }}
                  >
                    Show all runs
                  </Button>
                </Text>
              ) : (
                <EmptyState
                  icon={FiActivity}
                  title="No runs yet"
                  action={
                    <Button asChild size="xs" colorPalette="teal">
                      <RouterLink to="/agents">Go to agents</RouterLink>
                    </Button>
                  }
                >
                  Runs appear here when an agent is triggered by an event, a webhook or a schedule, or run by hand.
                </EmptyState>
              ))}

            {runs.length > 0 && (
              <Table.ScrollArea whiteSpace="nowrap" borderWidth="1px" borderRadius="md">
                <Table.Root size="sm" css={{ "& th, & td": { px: 3 } }}>
                  <Table.Header>
                    <Table.Row>
                      <Table.ColumnHeader>Started</Table.ColumnHeader>
                      <Table.ColumnHeader>Agent</Table.ColumnHeader>
                      <Table.ColumnHeader>Status</Table.ColumnHeader>
                      <Table.ColumnHeader>Trigger</Table.ColumnHeader>
                      <Table.ColumnHeader textAlign="end" title="Model calls / function calls">
                        Turns / calls
                      </Table.ColumnHeader>
                      <Table.ColumnHeader textAlign="end">Tokens</Table.ColumnHeader>
                      <Table.ColumnHeader textAlign="end">Cost</Table.ColumnHeader>
                      <Table.ColumnHeader textAlign="end">Duration</Table.ColumnHeader>
                    </Table.Row>
                  </Table.Header>
                  <Table.Body>
                    {runs.map((r) => {
                      const agent = agentOf(r);
                      const trigger = triggerSummary(r.triggerType, r.trigger);
                      return (
                        <Table.Row
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
                          <Table.Cell whiteSpace="nowrap" title={formatTime(r.startedAt)}>
                            {formatShortTime(r.startedAt)}
                          </Table.Cell>
                          <Table.Cell maxW="160px" overflow="hidden" textOverflow="ellipsis" fontWeight="medium" title={agent}>
                            {agent}
                          </Table.Cell>
                          <Table.Cell>
                            <RunStatusBadge run={r} />
                          </Table.Cell>
                          <Table.Cell maxW="180px" overflow="hidden" textOverflow="ellipsis" title={trigger}>
                            {trigger}
                          </Table.Cell>
                          <Table.Cell textAlign="end" whiteSpace="nowrap">
                            {r.turns} / {r.toolCalls}
                          </Table.Cell>
                          <Table.Cell textAlign="end">{formatTokens(r.inputTokens + r.outputTokens)}</Table.Cell>
                          <Table.Cell textAlign="end">{formatCost(r.costUsd)}</Table.Cell>
                          <Table.Cell textAlign="end">{formatDuration(r.durationMs)}</Table.Cell>
                        </Table.Row>
                      );
                    })}
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
          </Box>
        </Tabs.Content>
        <Tabs.Content value="changes" px={0} py={4}>
          <Box bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 5 }}>
            <Text fontWeight="semibold" mb={1}>
              Content changes
            </Text>
            <Text fontSize="sm" color="gray.500" mb={4}>
              Every change to an entry, by users, API tokens and agents. Open one to see its history and restore a version.
            </Text>
            <ContentChanges filters />
          </Box>
        </Tabs.Content>
      </Tabs.Root>

      <RunDetailDrawer
        agentName={open?.agent ?? ""}
        runId={open?.id ?? null}
        onClose={() => {
          setOpen(null);
          refetch();
        }}
      />
    </Page>
  );
}

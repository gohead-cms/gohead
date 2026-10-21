import { Link as RouterLink } from "react-router-dom";
import { Badge, Button, Table } from "@chakra-ui/react";
import { FiCpu } from "react-icons/fi";
import { EmptyState } from "../../../shared/ui/EmptyState";
import { formatCost, formatTime, statusColor } from "../../agents/components/runFormat";
import type { RecentRun } from "../overview";
import { timeAgo } from "../overview";

/** The last agent runs, whatever the agent. */
export function RecentRuns({ runs, hasAgents }: { runs: RecentRun[]; hasAgents: boolean }) {
  if (runs.length === 0) {
    return (
      <EmptyState
        icon={FiCpu}
        title="No agent runs yet"
        action={
          <Button asChild size="xs" colorPalette="purple">
            <RouterLink to={hasAgents ? "/agents" : "/agents/new"}>{hasAgents ? "Go to agents" : "Create an agent"}</RouterLink>
          </Button>
        }
      >
        {hasAgents ? "Runs appear here when an agent is triggered." : "Agents react to content changes, webhooks or a schedule."}
      </EmptyState>
    );
  }
  return (
    <Table.ScrollArea>
      <Table.Root size="sm">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>Agent</Table.ColumnHeader>
            <Table.ColumnHeader>Status</Table.ColumnHeader>
            <Table.ColumnHeader>Trigger</Table.ColumnHeader>
            <Table.ColumnHeader textAlign="end">Cost</Table.ColumnHeader>
            <Table.ColumnHeader>Started</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {runs.map((r) => (
            <Table.Row key={r.id}>
              <Table.Cell fontWeight="medium">{r.agentName || `Agent #${r.agentId}`}</Table.Cell>
              <Table.Cell>
                <Badge colorPalette={statusColor[r.status] ?? "gray"}>{r.status}</Badge>
              </Table.Cell>
              <Table.Cell color="gray.600">{r.triggerType.replace(/_/g, " ")}</Table.Cell>
              <Table.Cell textAlign="end">{formatCost(r.costUsd)}</Table.Cell>
              <Table.Cell title={formatTime(r.startedAt)} color="gray.600">
                {timeAgo(r.startedAt)}
              </Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </Table.ScrollArea>
  );
}

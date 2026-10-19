import { Link as RouterLink } from "react-router-dom";
import { Badge, Button, Table, TableContainer, Tbody, Td, Th, Thead, Tr } from "@chakra-ui/react";
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
          <Button as={RouterLink} to={hasAgents ? "/agents" : "/agents/new"} size="sm" colorScheme="purple">
            {hasAgents ? "Go to agents" : "Create an agent"}
          </Button>
        }
      >
        {hasAgents ? "Runs appear here when an agent is triggered." : "Agents react to content changes, webhooks or a schedule."}
      </EmptyState>
    );
  }
  return (
    <TableContainer>
      <Table size="sm">
        <Thead>
          <Tr>
            <Th>Agent</Th>
            <Th>Status</Th>
            <Th>Trigger</Th>
            <Th isNumeric>Cost</Th>
            <Th>Started</Th>
          </Tr>
        </Thead>
        <Tbody>
          {runs.map((r) => (
            <Tr key={r.id}>
              <Td fontWeight="medium">{r.agentName || `Agent #${r.agentId}`}</Td>
              <Td>
                <Badge colorScheme={statusColor[r.status] ?? "gray"}>{r.status}</Badge>
              </Td>
              <Td color="gray.600">{r.triggerType.replace(/_/g, " ")}</Td>
              <Td isNumeric>{formatCost(r.costUsd)}</Td>
              <Td title={formatTime(r.startedAt)} color="gray.600">
                {timeAgo(r.startedAt)}
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    </TableContainer>
  );
}

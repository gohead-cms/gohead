import { Link as RouterLink } from "react-router-dom";
import { Badge, Button, Flex, HStack, List, Text, VStack } from "@chakra-ui/react";
import { FiArrowRight } from "react-icons/fi";
import { ProposalBadges } from "./ProposalDrawer";
import { approvalsLink, proposalEntryLabel, proposalStatusColor } from "./proposals";
import { useRunProposals } from "./useProposals";

/**
 * The changes a run proposed (agents that need approval), with a link to
 * review them on the Approvals page. Renders nothing when there are none.
 */
export function RunProposals({ runId }: { runId: number }) {
  const { proposals, total, error } = useRunProposals(runId);
  if (error) {
    return (
      <Text fontSize="sm" color="red.500">
        {error}
      </Text>
    );
  }
  if (total === 0) return null;
  const pending = proposals.filter((p) => p.status === "pending").length;
  return (
    <VStack align="stretch" gap={2} data-testid="run-proposals">
      <Flex justify="space-between" align="center" gap={3} wrap="wrap">
        <Text fontWeight="bold" fontSize="xs" textTransform="uppercase" color="gray.500">
          Proposed changes ({total})
        </Text>
        <Button asChild size="2xs" colorPalette="purple" variant="outline">
          <RouterLink to={approvalsLink({ status: "all", agentRun: runId })}>
            {pending > 0 ? "Review in Approvals" : "Open in Approvals"}
            <FiArrowRight />
          </RouterLink>
        </Button>
      </Flex>
      <List.Root variant="plain" gap={1.5}>
        {proposals.map((p) => (
          <List.Item key={p.id} fontSize="sm">
            <HStack gap={2} wrap="wrap">
              <Badge colorPalette={proposalStatusColor[p.status] ?? "gray"} variant="outline">
                {p.status}
              </Badge>
              <ProposalBadges proposal={p} />
              <Text>{proposalEntryLabel(p)}</Text>
              {p.reason && (
                <Text color="gray.500" lineClamp={1} title={p.reason}>
                  — {p.reason}
                </Text>
              )}
            </HStack>
          </List.Item>
        ))}
      </List.Root>
      {total > proposals.length && (
        <Text fontSize="xs" color="gray.500">
          And {total - proposals.length} more.
        </Text>
      )}
    </VStack>
  );
}

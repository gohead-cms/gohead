import { Badge, SimpleGrid, Stat, StatLabel, StatNumber, Text, Wrap, WrapItem } from "@chakra-ui/react";
import { formatCost, formatTokens, statusColor } from "../../agents/components/runFormat";
import type { Overview } from "../overview";
import { successRate } from "../overview";

/** Agent runs of the last 24 hours: by status, tokens and cost. */
export function AgentActivityCard({ runs }: { runs: Overview["runs24h"] }) {
  if (runs.total === 0) {
    return <Text fontSize="sm" color="gray.500">No agent runs in the last 24 hours.</Text>;
  }
  const byStatus = [
    { status: "succeeded", n: runs.succeeded },
    { status: "failed", n: runs.failed },
    { status: "limited", n: runs.limited },
    { status: "running", n: runs.running },
  ] as const;
  return (
    <>
      <SimpleGrid columns={2} spacingX={4} spacingY={3} mb={4}>
        <Stat size="sm">
          <StatLabel color="gray.500">Runs</StatLabel>
          <StatNumber>{runs.total.toLocaleString()}</StatNumber>
        </Stat>
        <Stat size="sm">
          <StatLabel color="gray.500">Success rate</StatLabel>
          <StatNumber>{successRate(runs)}</StatNumber>
        </Stat>
        <Stat size="sm">
          <StatLabel color="gray.500">Tokens in / out</StatLabel>
          <StatNumber>
            {formatTokens(runs.inputTokens)} / {formatTokens(runs.outputTokens)}
          </StatNumber>
        </Stat>
        <Stat size="sm">
          <StatLabel color="gray.500">Cost</StatLabel>
          <StatNumber>{formatCost(runs.costUsd)}</StatNumber>
        </Stat>
      </SimpleGrid>
      <Wrap spacing={2}>
        {byStatus
          .filter((s) => s.n > 0)
          .map((s) => (
            <WrapItem key={s.status}>
              <Badge colorScheme={statusColor[s.status]} variant={s.status === "failed" ? "solid" : "subtle"}>
                {s.n} {s.status}
              </Badge>
            </WrapItem>
          ))}
      </Wrap>
    </>
  );
}

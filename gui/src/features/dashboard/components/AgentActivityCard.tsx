import { Badge, SimpleGrid, Stat, Text, Wrap, WrapItem } from "@chakra-ui/react";
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
      <SimpleGrid columns={2} columnGap={4} rowGap={3} mb={4}>
        <Stat.Root size="sm">
          <Stat.Label color="gray.500">Runs</Stat.Label>
          <Stat.ValueText>{runs.total.toLocaleString()}</Stat.ValueText>
        </Stat.Root>
        <Stat.Root size="sm">
          <Stat.Label color="gray.500">Success rate</Stat.Label>
          <Stat.ValueText>{successRate(runs)}</Stat.ValueText>
        </Stat.Root>
        <Stat.Root size="sm">
          <Stat.Label color="gray.500">Tokens in / out</Stat.Label>
          <Stat.ValueText>
            {formatTokens(runs.inputTokens)} / {formatTokens(runs.outputTokens)}
          </Stat.ValueText>
        </Stat.Root>
        <Stat.Root size="sm">
          <Stat.Label color="gray.500">Cost</Stat.Label>
          <Stat.ValueText>{formatCost(runs.costUsd)}</Stat.ValueText>
        </Stat.Root>
      </SimpleGrid>
      <Wrap gap={2}>
        {byStatus
          .filter((s) => s.n > 0)
          .map((s) => (
            <WrapItem key={s.status}>
              <Badge colorPalette={statusColor[s.status]} variant={s.status === "failed" ? "solid" : "subtle"}>
                {s.n} {s.status}
              </Badge>
            </WrapItem>
          ))}
      </Wrap>
    </>
  );
}

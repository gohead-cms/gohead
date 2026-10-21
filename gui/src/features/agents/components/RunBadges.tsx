import { Badge, HStack } from "@chakra-ui/react";
import { type AgentRun } from "../../../shared/types";
import { statusColor } from "./runFormat";

/** A run's status, and "test run" for a dry run. */
export function RunStatusBadge({ run }: { run: Pick<AgentRun, "status" | "error" | "dryRun"> }) {
  return (
    <HStack gap={1.5} display="inline-flex">
      <Badge
        colorPalette={statusColor[run.status] ?? "gray"}
        variant={run.status === "queued" ? "outline" : "subtle"}
        title={run.error || (run.status === "queued" ? "Waiting for a worker" : undefined)}
      >
        {run.status}
      </Badge>
      {run.dryRun && <DryRunBadge />}
    </HStack>
  );
}

/** Marks a test run: nothing it did was written. */
export function DryRunBadge() {
  return (
    <Badge colorPalette="purple" variant="outline" title="Test run: content changes were only simulated">
      test run
    </Badge>
  );
}

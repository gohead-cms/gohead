import { useState } from "react";
import { Badge, Box, Button, Flex, HStack, List, Text, VStack } from "@chakra-ui/react";
import { Checkbox } from "@/components/ui/checkbox";
import { FiRotateCcw } from "react-icons/fi";
import { api, ApiError } from "../../services/api";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { useMe } from "../auth/me";
import { ContentChanges } from "./ContentChanges";

/** What reverting a run did to one entry. */
interface RevertOutcome {
  entityType: "item" | "singleton";
  entityId: number;
  typeName: string;
  result: "reverted" | "deleted" | "skipped" | "failed";
  version?: number;
  reason?: string;
}

const resultColor: Record<RevertOutcome["result"], string> = {
  reverted: "green",
  deleted: "orange",
  skipped: "gray",
  failed: "red",
};

/**
 * The content an agent run changed, and (admins) reverting those changes:
 * entries go back to their state before the run, and entries the run
 * created are deleted.
 */
export function RunContentChanges({ agentName, runId }: { agentName: string; runId: number }) {
  const { isAdmin } = useMe();
  const notify = useNotify();
  const [force, setForce] = useState(false);
  const [outcomes, setOutcomes] = useState<RevertOutcome[] | null>(null);
  // Bumped after a revert so the list reloads.
  const [version, setVersion] = useState(0);

  const revert = async () => {
    try {
      const res = await api<RevertOutcome[]>(
        "POST",
        `/admin/agents/${encodeURIComponent(agentName)}/runs/${runId}/revert?force=${force}`,
      );
      const list = Array.isArray(res.data) ? res.data : [];
      setOutcomes(list);
      const counts: Record<string, number> = {};
      for (const o of list) counts[o.result] = (counts[o.result] ?? 0) + 1;
      Object.assign(counts, (res.meta?.counts ?? {}) as Record<string, number>);
      const summary = (["reverted", "deleted", "skipped", "failed"] as const)
        .filter((k) => counts[k])
        .map((k) => `${counts[k]} ${k}`)
        .join(", ");
      if ((counts.reverted ?? 0) + (counts.deleted ?? 0) === 0) {
        notify.warning("Nothing was reverted — see details", summary || undefined);
      } else {
        notify.success("Run reverted", summary);
      }
      setVersion((v) => v + 1);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        // The run is still going, or it changed no content.
        notify.failure(err, /still running/i.test(err.message) ? "The run is still running: revert it once it ends" : "Nothing to revert");
      } else {
        notify.failure(err, "Could not revert the run");
      }
    }
  };

  return (
    <VStack align="stretch" gap={3}>
      <Flex justify="space-between" align="center" gap={3} wrap="wrap">
        <Text fontWeight="bold" fontSize="xs" textTransform="uppercase" color="gray.500">
          Content changes
        </Text>
        {isAdmin && (
          <ConfirmButton
            trigger={
              <Button size="2xs" variant="outline" colorPalette="red">
                <FiRotateCcw />
                Revert content changes
              </Button>
            }
            title={`Revert the content changes of run #${runId}?`}
            body={
              <VStack align="stretch" gap={3}>
                <Text>
                  Every entry this run changed is restored to its state before the run, and entries the run created are deleted.
                  Each revert is recorded as a new version, so it can be undone from the entry's history.
                </Text>
                <Text fontSize="sm" color="gray.500">
                  Entries someone changed after the run are skipped, unless you check the box below.
                </Text>
                <Checkbox size="sm" checked={force} onCheckedChange={(e) => setForce(!!e.checked)} colorPalette="red">
                  Also revert entries changed by someone after the run
                </Checkbox>
              </VStack>
            }
            confirmLabel="Revert"
            onConfirm={revert}
            // Each revert starts unforced, whether the last was confirmed or cancelled.
            onClose={() => setForce(false)}
          />
        )}
      </Flex>

      {outcomes && (
        <Box borderWidth="1px" borderRadius="md" p={3} data-testid="revert-outcomes">
          <Text fontSize="sm" fontWeight="semibold" mb={2}>
            Revert result
          </Text>
          {outcomes.length === 0 ? (
            <Text fontSize="sm" color="gray.500">
              No entry to revert.
            </Text>
          ) : (
            <List.Root variant="plain" gap={1.5}>
              {outcomes.map((o) => (
                <List.Item key={`${o.entityType}-${o.entityId}`} fontSize="sm">
                  <HStack gap={2} align="start">
                    <Badge colorPalette={resultColor[o.result] ?? "gray"}>{o.result}</Badge>
                    <Text>
                      <Text as="span" fontWeight="medium">
                        {o.typeName}
                      </Text>
                      {o.entityType === "item" && ` #${o.entityId}`}
                      {o.version !== undefined && (
                        <Text as="span" color="gray.500">
                          {" "}
                          (to v{o.version})
                        </Text>
                      )}
                      {o.reason && (
                        <Text as="span" color="gray.500">
                          {" "}
                          — {o.reason}
                        </Text>
                      )}
                    </Text>
                  </HStack>
                </List.Item>
              ))}
            </List.Root>
          )}
        </Box>
      )}

      <ContentChanges key={version} fixed={{ agentRun: runId }} pageSize={10} emptyText="This run did not change any content." />
    </VStack>
  );
}

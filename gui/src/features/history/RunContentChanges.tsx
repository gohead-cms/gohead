import { useState } from "react";
import {
  Badge,
  Box,
  Button,
  Checkbox,
  Flex,
  HStack,
  List,
  ListItem,
  Text,
  VStack,
} from "@chakra-ui/react";
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
      const counts = (res.meta?.counts ?? {}) as Record<string, number>;
      const summary = Object.entries(counts)
        .map(([k, n]) => `${n} ${k}`)
        .join(", ");
      notify.success("Run reverted", summary || undefined);
      setForce(false);
      setVersion((v) => v + 1);
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        notify.failure(err, "Nothing to revert");
      } else {
        notify.failure(err, "Could not revert the run");
      }
    }
  };

  return (
    <VStack align="stretch" spacing={3}>
      <Flex justify="space-between" align="center" gap={3} wrap="wrap">
        <Text fontWeight="bold" fontSize="xs" textTransform="uppercase" color="gray.500">
          Content changes
        </Text>
        {isAdmin && (
          <ConfirmButton
            trigger={
              <Button size="xs" variant="outline" colorScheme="red" leftIcon={<FiRotateCcw />}>
                Revert content changes
              </Button>
            }
            title={`Revert the content changes of run #${runId}?`}
            body={
              <VStack align="stretch" spacing={3}>
                <Text>
                  Every entry this run changed is restored to its state before the run, and entries the run created are deleted.
                  Each revert is recorded as a new version, so it can be undone from the entry's history.
                </Text>
                <Text fontSize="sm" color="gray.500">
                  Entries someone changed after the run are skipped, unless you check the box below.
                </Text>
                <Checkbox isChecked={force} onChange={(e) => setForce(e.target.checked)} colorScheme="red">
                  Also revert entries changed by someone after the run
                </Checkbox>
              </VStack>
            }
            confirmLabel="Revert"
            onConfirm={revert}
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
            <List spacing={1.5}>
              {outcomes.map((o) => (
                <ListItem key={`${o.entityType}-${o.entityId}`} fontSize="sm">
                  <HStack spacing={2} align="start">
                    <Badge colorScheme={resultColor[o.result] ?? "gray"}>{o.result}</Badge>
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
                </ListItem>
              ))}
            </List>
          )}
        </Box>
      )}

      <ContentChanges key={version} fixed={{ agentRun: runId }} pageSize={10} emptyText="This run did not change any content." />
    </VStack>
  );
}

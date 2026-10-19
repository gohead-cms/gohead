import { useState } from "react";
import {
  Alert,
  AlertIcon,
  Button,
  FormControl,
  FormHelperText,
  FormLabel,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Radio,
  RadioGroup,
  Select,
  Spinner,
  Stack,
  Text,
  Textarea,
  VStack,
  useDisclosure,
} from "@chakra-ui/react";
import { FiZap } from "react-icons/fi";
import { api } from "../../../services/api";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import { useNotify } from "../../../shared/hooks/useNotify";
import { type AgentNodeData } from "../../../shared/types";
import { errorMessage } from "../../../shared/utils/errors";
import { dataTitle } from "../../history/history";
import { EVENTS } from "./agentForm";

/** The response of POST /admin/agents/{name}/run and /test. */
export interface QueuedRun {
  status: "queued";
  runId: number;
  dryRun: boolean;
}

const ITEM_CHOICES = 20;

/** The latest entries of a collection, to simulate an event on one. */
function useRecentItems(collection: string | null) {
  const path = collection ? `/api/collections/${encodeURIComponent(collection)}?status=all&sort=id:desc&pageSize=${ITEM_CHOICES}` : null;
  const { data, loading, error } = useApiQuery(path, async () => {
    const json = await getJSON<Record<string, unknown>[]>(path!, "Failed to load entries");
    return Array.isArray(json.data) ? json.data : [];
  });
  return { items: data ?? [], loading, error };
}

const eventLabel = (value: string) => EVENTS.find((e) => e.value === value)?.label ?? value;

/**
 * Queues a test run (POST /admin/agents/{name}/test): the agent runs for
 * real, but content functions only report what they would change. A
 * collection-event agent can be tried on an existing entry.
 */
export function TestRunButton({ agent, onQueued }: { agent: AgentNodeData; onQueued: (runId: number) => void }) {
  const { isOpen, onOpen, onClose } = useDisclosure();
  const trigger = agent.schema.trigger;
  const collection = trigger.type === "collection_event" ? (trigger.event_trigger?.collection ?? "") : "";
  const events = trigger.event_trigger?.events ?? [];
  const [chosenMode, setMode] = useState<"event" | "input">(collection ? "event" : "input");
  // The agent may have changed since (no collection trigger, other events).
  const mode = collection ? chosenMode : "input";
  const [itemId, setItemId] = useState("");
  const [chosenEvent, setEvent] = useState(events[0] ?? "");
  const event = events.includes(chosenEvent) ? chosenEvent : (events[0] ?? "");
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const notify = useNotify();
  const { items, loading: itemsLoading, error: itemsError } = useRecentItems(isOpen && mode === "event" && collection ? collection : null);
  const chosenItem = itemId && items.some((it) => String(it.id) === itemId) ? itemId : items[0] ? String(items[0].id) : "";

  const close = () => {
    setError(null);
    onClose();
  };

  const run = async () => {
    setBusy(true);
    setError(null);
    const body = mode === "event" ? { item_id: Number(chosenItem), event: event || undefined } : { input: input.trim() };
    try {
      const res = await api<QueuedRun>("POST", `/admin/agents/${encodeURIComponent(agent.name)}/test`, body);
      notify.success("Test run queued", "Nothing it does is written: content changes are only reported.");
      onClose();
      if (res.data?.runId) onQueued(res.data.runId);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const canRun = mode === "input" || Boolean(chosenItem);

  return (
    <>
      <Button size="sm" variant="outline" colorScheme="purple" leftIcon={<FiZap />} onClick={onOpen}>
        Test run
      </Button>
      <Modal isOpen={isOpen} onClose={close} size="lg">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Test {agent.name}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack align="stretch" spacing={4}>
              <Text fontSize="sm" color="gray.600">
                A dry run: the agent calls the model and reads content as usual, but its creates, updates and deletes are only reported (“would
                update…”). Nothing is written. It counts toward the agent's limits.
              </Text>
              {collection && (
                <RadioGroup value={mode} onChange={(v) => setMode(v as "event" | "input")}>
                  <Stack direction={{ base: "column", sm: "row" }} spacing={5}>
                    <Radio value="event">On an entry of {collection}</Radio>
                    <Radio value="input">With a message</Radio>
                  </Stack>
                </RadioGroup>
              )}
              {mode === "event" ? (
                <>
                  <FormControl isRequired>
                    <FormLabel htmlFor="test-run-item" fontSize="sm">
                      Entry
                    </FormLabel>
                    {itemsLoading && items.length === 0 ? (
                      <Spinner size="sm" />
                    ) : items.length === 0 ? (
                      <Text fontSize="sm" color="gray.500">
                        {itemsError ?? `${collection} has no entries yet: create one, or test with a message.`}
                      </Text>
                    ) : (
                      <Select id="test-run-item" value={chosenItem} onChange={(e) => setItemId(e.target.value)}>
                        {items.map((it) => {
                          const title = dataTitle(it);
                          return (
                            <option key={String(it.id)} value={String(it.id)}>
                              #{String(it.id)}
                              {title ? ` — ${title}` : ""}
                            </option>
                          );
                        })}
                      </Select>
                    )}
                    <FormHelperText>The {ITEM_CHOICES} latest entries. The agent gets the entry's data as in a real event.</FormHelperText>
                  </FormControl>
                  {events.length > 0 && (
                    <FormControl>
                      <FormLabel htmlFor="test-run-event" fontSize="sm">
                        Event
                      </FormLabel>
                      <Select id="test-run-event" value={event} onChange={(e) => setEvent(e.target.value)}>
                        {events.map((ev) => (
                          <option key={ev} value={ev}>
                            {eventLabel(ev)} ({ev})
                          </option>
                        ))}
                      </Select>
                    </FormControl>
                  )}
                </>
              ) : (
                <FormControl>
                  <FormLabel htmlFor="test-run-input" fontSize="sm">
                    Input (optional)
                  </FormLabel>
                  <Textarea
                    id="test-run-input"
                    rows={5}
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    placeholder="The first message sent to the model"
                  />
                  <FormHelperText>Empty: “This is a test run started from the admin UI. Do your task.”</FormHelperText>
                </FormControl>
              )}
              {error && (
                <Alert status="error" borderRadius="md">
                  <AlertIcon />
                  {error}
                </Alert>
              )}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={close} isDisabled={busy}>
              Cancel
            </Button>
            <Button colorScheme="purple" onClick={run} isLoading={busy} loadingText="Queuing" isDisabled={!canRun}>
              Start test run
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  );
}

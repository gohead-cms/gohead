import { useState } from "react";
import { Button, Field, NativeSelect, Spinner, Stack, Text, Textarea, VStack, useDisclosure } from "@chakra-ui/react";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { Radio, RadioGroup } from "@/components/ui/radio";
import { FiZap } from "react-icons/fi";
import { api } from "../../../services/api";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import { useNotify } from "../../../shared/hooks/useNotify";
import { type AgentNodeData } from "../../../shared/types";
import { errorMessage } from "../../../shared/utils/errors";
import { dataTitle } from "../../history/history";
import { EVENTS } from "./agentForm";
import { Notice } from "./EditorParts";

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
  const { open, onOpen, onClose } = useDisclosure();
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
  const { items, loading: itemsLoading, error: itemsError } = useRecentItems(open && mode === "event" && collection ? collection : null);
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
      <Button size="xs" variant="outline" colorPalette="purple" onClick={onOpen}>
        <FiZap />
        Test run
      </Button>
      <DialogRoot
        open={open}
        onOpenChange={(e) => {
          if (!e.open) close();
        }}
        size="lg"
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Test {agent.name}</DialogTitle>
          </DialogHeader>
          <DialogBody>
            <VStack align="stretch" gap={4}>
              <Text fontSize="sm" color="gray.600">
                A dry run: the agent calls the model and reads content as usual, but its creates, updates and deletes are only reported (“would
                update…”). Nothing is written. It counts toward the agent's limits.
              </Text>
              {collection && (
                <RadioGroup size="sm" value={mode} onValueChange={(e) => setMode(e.value === "event" ? "event" : "input")}>
                  <Stack direction={{ base: "column", sm: "row" }} gap={5}>
                    <Radio value="event">On an entry of {collection}</Radio>
                    <Radio value="input">With a message</Radio>
                  </Stack>
                </RadioGroup>
              )}
              {mode === "event" ? (
                <>
                  <Field.Root required id="test-run-item" gap={2}>
                    <Field.Label>
                      Entry
                      <Field.RequiredIndicator />
                    </Field.Label>
                    {itemsLoading && items.length === 0 ? (
                      <Spinner size="sm" />
                    ) : items.length === 0 ? (
                      <Text fontSize="sm" color="gray.500">
                        {itemsError ?? `${collection} has no entries yet: create one, or test with a message.`}
                      </Text>
                    ) : (
                      <NativeSelect.Root>
                        <NativeSelect.Field value={chosenItem} onChange={(e) => setItemId(e.target.value)}>
                          {items.map((it) => {
                            const title = dataTitle(it);
                            return (
                              <option key={String(it.id)} value={String(it.id)}>
                                #{String(it.id)}
                                {title ? ` — ${title}` : ""}
                              </option>
                            );
                          })}
                        </NativeSelect.Field>
                        <NativeSelect.Indicator />
                      </NativeSelect.Root>
                    )}
                    <Field.HelperText fontSize="sm">The {ITEM_CHOICES} latest entries. The agent gets the entry's data as in a real event.</Field.HelperText>
                  </Field.Root>
                  {events.length > 0 && (
                    <Field.Root id="test-run-event" gap={2}>
                      <Field.Label>Event</Field.Label>
                      <NativeSelect.Root>
                        <NativeSelect.Field value={event} onChange={(e) => setEvent(e.target.value)}>
                          {events.map((ev) => (
                            <option key={ev} value={ev}>
                              {eventLabel(ev)} ({ev})
                            </option>
                          ))}
                        </NativeSelect.Field>
                        <NativeSelect.Indicator />
                      </NativeSelect.Root>
                    </Field.Root>
                  )}
                </>
              ) : (
                <Field.Root id="test-run-input" gap={2}>
                  <Field.Label>Input (optional)</Field.Label>
                  <Textarea
                    rows={5}
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    placeholder="The first message sent to the model"
                  />
                  <Field.HelperText fontSize="sm">Empty: “This is a test run started from the admin UI. Do your task.”</Field.HelperText>
                </Field.Root>
              )}
              {error && (
                <Notice status="error" borderRadius="md">
                  {error}
                </Notice>
              )}
            </VStack>
          </DialogBody>
          <DialogFooter>
            <Button variant="ghost" onClick={close} disabled={busy}>
              Cancel
            </Button>
            <Button colorPalette="purple" onClick={run} loading={busy} loadingText="Queuing" disabled={!canRun}>
              Start test run
            </Button>
          </DialogFooter>
          <DialogCloseTrigger />
        </DialogContent>
      </DialogRoot>
    </>
  );
}

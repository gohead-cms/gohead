import { useState } from "react";
import { Button, Field, Textarea, useDisclosure } from "@chakra-ui/react";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { FiPlay } from "react-icons/fi";
import { api } from "../../../services/api";
import { useNotify } from "../../../shared/hooks/useNotify";
import { errorMessage } from "../../../shared/utils/errors";
import { Notice } from "./EditorParts";
import { type QueuedRun } from "./TestRunButton";

/** Queues a manual run (POST /admin/agents/{name}/run) with an optional input; onQueued gets the run to follow. */
export function RunNowButton({ agentName, onQueued }: { agentName: string; onQueued: (runId?: number) => void }) {
  const { open, onOpen, onClose } = useDisclosure();
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const notify = useNotify();

  const close = () => {
    setError(null);
    onClose();
  };

  const run = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await api<QueuedRun>("POST", `/admin/agents/${encodeURIComponent(agentName)}/run`, { input: input.trim() });
      notify.success("Run queued", "It starts as soon as a worker picks it up.");
      setInput("");
      onClose();
      onQueued(res.data?.runId);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <Button size="xs" colorPalette="teal" onClick={onOpen}>
        <FiPlay />
        Run now
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
            <DialogTitle>Run {agentName} now</DialogTitle>
          </DialogHeader>
          <DialogCloseTrigger />
          <DialogBody>
            <Field.Root gap={2}>
              <Field.Label fontSize="md">Input (optional)</Field.Label>
              <Textarea rows={5} value={input} onChange={(e) => setInput(e.target.value)} placeholder="The first message sent to the model" />
              <Field.HelperText fontSize="sm">
                Empty: “This is a manual run started from the admin UI. Do your task.” The run counts toward the agent's limits.
              </Field.HelperText>
            </Field.Root>
            {error && (
              <Notice status="error" mt={4} borderRadius="md">
                {error}
              </Notice>
            )}
          </DialogBody>
          <DialogFooter>
            <Button variant="ghost" onClick={close} disabled={busy}>
              Cancel
            </Button>
            <Button colorPalette="teal" onClick={run} loading={busy} loadingText="Queuing">
              Queue run
            </Button>
          </DialogFooter>
        </DialogContent>
      </DialogRoot>
    </>
  );
}

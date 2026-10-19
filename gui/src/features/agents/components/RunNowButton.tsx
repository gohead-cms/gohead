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
  Textarea,
  useDisclosure,
} from "@chakra-ui/react";
import { FiPlay } from "react-icons/fi";
import { api } from "../../../services/api";
import { useNotify } from "../../../shared/hooks/useNotify";
import { errorMessage } from "../../../shared/utils/errors";
import { type QueuedRun } from "./TestRunButton";

/** Queues a manual run (POST /admin/agents/{name}/run) with an optional input; onQueued gets the run to follow. */
export function RunNowButton({ agentName, onQueued }: { agentName: string; onQueued: (runId?: number) => void }) {
  const { isOpen, onOpen, onClose } = useDisclosure();
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
      <Button size="sm" colorScheme="teal" leftIcon={<FiPlay />} onClick={onOpen}>
        Run now
      </Button>
      <Modal isOpen={isOpen} onClose={close} size="lg">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Run {agentName} now</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl>
              <FormLabel>Input (optional)</FormLabel>
              <Textarea rows={5} value={input} onChange={(e) => setInput(e.target.value)} placeholder="The first message sent to the model" />
              <FormHelperText>
                Empty: “This is a manual run started from the admin UI. Do your task.” The run counts toward the agent's limits.
              </FormHelperText>
            </FormControl>
            {error && (
              <Alert status="error" mt={4} borderRadius="md">
                <AlertIcon />
                {error}
              </Alert>
            )}
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={close} isDisabled={busy}>
              Cancel
            </Button>
            <Button colorScheme="teal" onClick={run} isLoading={busy} loadingText="Queuing">
              Queue run
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  );
}

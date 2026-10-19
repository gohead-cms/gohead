import { useRef, useState } from "react";
import {
  AlertDialog,
  AlertDialogBody,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogOverlay,
  Button,
} from "@chakra-ui/react";
import { api } from "../../../services/api";
import { useNotify } from "../../../shared/hooks/useNotify";
import { serverMessage } from "../../model/api";
import type { NodeRef } from "../graph";

/** Confirms and deletes a collection or an agent picked on the canvas. */
export function DeleteNodeDialog({ target, onClose, onDeleted }: { target: NodeRef | null; onClose: () => void; onDeleted: () => void }) {
  const cancelRef = useRef<HTMLButtonElement>(null);
  const notify = useNotify();
  const [busy, setBusy] = useState(false);

  const confirm = async () => {
    if (!target) return;
    setBusy(true);
    try {
      const base = target.kind === "collection" ? "/admin/collections/" : "/admin/agents/";
      await api("DELETE", base + encodeURIComponent(target.name));
      notify.success(`Deleted ${target.kind} "${target.name}"`);
      onDeleted();
    } catch (err) {
      notify.failure(new Error(serverMessage(err)), `Could not delete "${target.name}"`);
    } finally {
      setBusy(false);
      onClose();
    }
  };

  return (
    <AlertDialog isOpen={!!target} leastDestructiveRef={cancelRef} onClose={onClose}>
      <AlertDialogOverlay>
        <AlertDialogContent>
          <AlertDialogHeader fontSize="lg" fontWeight="bold">
            Delete {target?.kind} "{target?.name}"?
          </AlertDialogHeader>
          <AlertDialogBody>
            {target?.kind === "collection"
              ? "The collection and all its items are deleted. This cannot be undone."
              : "The agent and its configuration are deleted. This cannot be undone."}
          </AlertDialogBody>
          <AlertDialogFooter>
            <Button ref={cancelRef} onClick={onClose} isDisabled={busy}>
              Cancel
            </Button>
            <Button colorScheme="red" onClick={confirm} ml={3} isLoading={busy}>
              Delete
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialogOverlay>
    </AlertDialog>
  );
}

import { type ReactElement, cloneElement, useRef, useState } from "react";
import { Button, useDisclosure } from "@chakra-ui/react";
import {
  DialogBody,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogRoot,
  DialogTitle,
} from "@/components/ui/dialog";

/**
 * Wraps a trigger (a Button or IconButton) with a confirmation dialog.
 * onConfirm may be async; the dialog stays open while it runs.
 */
export function ConfirmButton({
  trigger,
  title,
  body,
  confirmLabel = "Delete",
  confirmColorScheme = "red",
  onConfirm,
  onClose: onClosed,
}: {
  trigger: ReactElement<{ onClick?: () => void }>;
  title: string;
  body: React.ReactNode;
  confirmLabel?: string;
  /** The confirm button's color: red for destructive actions. */
  confirmColorScheme?: string;
  onConfirm: () => Promise<void> | void;
  /** Called whenever the dialog closes (cancelled or confirmed). */
  onClose?: () => void;
}) {
  const { open, onOpen, onClose } = useDisclosure({ onClose: onClosed });
  const cancelRef = useRef<HTMLButtonElement>(null);
  const [busy, setBusy] = useState(false);

  const confirm = async () => {
    setBusy(true);
    try {
      await onConfirm();
      onClose();
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      {cloneElement(trigger, { onClick: onOpen })}
      <DialogRoot
        role="alertdialog"
        // As v2's AlertDialog: a click outside cancels.
        closeOnInteractOutside
        open={open}
        onOpenChange={(e) => {
          if (!e.open) onClose();
        }}
        initialFocusEl={() => cancelRef.current}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle fontSize="lg" fontWeight="bold">
              {title}
            </DialogTitle>
          </DialogHeader>
          <DialogBody>{body}</DialogBody>
          <DialogFooter>
            <Button ref={cancelRef} variant="subtle" onClick={onClose} disabled={busy}>
              Cancel
            </Button>
            <Button colorPalette={confirmColorScheme} onClick={confirm} ml={3} loading={busy}>
              {confirmLabel}
            </Button>
          </DialogFooter>
        </DialogContent>
      </DialogRoot>
    </>
  );
}

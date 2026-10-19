import { ReactElement, cloneElement, useRef, useState } from "react";
import {
  AlertDialog,
  AlertDialogBody,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogOverlay,
  Button,
  useDisclosure,
} from "@chakra-ui/react";

/**
 * Wraps a trigger (a Button or IconButton) with a confirmation dialog.
 * onConfirm may be async; the dialog stays open while it runs.
 */
export function ConfirmButton({
  trigger,
  title,
  body,
  confirmLabel = "Delete",
  onConfirm,
}: {
  trigger: ReactElement<{ onClick?: () => void }>;
  title: string;
  body: React.ReactNode;
  confirmLabel?: string;
  onConfirm: () => Promise<void> | void;
}) {
  const { isOpen, onOpen, onClose } = useDisclosure();
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
      <AlertDialog isOpen={isOpen} leastDestructiveRef={cancelRef} onClose={onClose}>
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {title}
            </AlertDialogHeader>
            <AlertDialogBody>{body}</AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={onClose} isDisabled={busy}>
                Cancel
              </Button>
              <Button colorScheme="red" onClick={confirm} ml={3} isLoading={busy}>
                {confirmLabel}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </>
  );
}

import { useCallback, useRef, useState } from "react";
import {
  AlertDialog,
  AlertDialogBody,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogOverlay,
  Button,
} from "@chakra-ui/react";

/**
 * Switching the locale of an editor replaces its form: with unsaved edits,
 * the switch asks first. Forms call markDirty when they change and
 * markClean once saved; request(code) switches, or asks when dirty.
 */
export function useLocaleGuard(onSwitch: (code: string) => void) {
  const dirty = useRef(false);
  const [pending, setPending] = useState<string | null>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);

  const markDirty = useCallback(() => {
    dirty.current = true;
  }, []);
  const markClean = useCallback(() => {
    dirty.current = false;
  }, []);

  const request = (code: string) => {
    if (dirty.current) {
      setPending(code);
      return;
    }
    onSwitch(code);
  };

  const discard = () => {
    if (pending !== null) {
      dirty.current = false;
      onSwitch(pending);
    }
    setPending(null);
  };

  const dialog = (
    <AlertDialog isOpen={pending !== null} leastDestructiveRef={cancelRef} onClose={() => setPending(null)}>
      <AlertDialogOverlay>
        <AlertDialogContent>
          <AlertDialogHeader fontSize="lg" fontWeight="bold">
            Discard unsaved changes?
          </AlertDialogHeader>
          <AlertDialogBody>
            The changes made in this locale are not saved. Switching to {pending} loses them; stay to save them first.
          </AlertDialogBody>
          <AlertDialogFooter>
            <Button ref={cancelRef} onClick={() => setPending(null)}>
              Keep editing
            </Button>
            <Button colorScheme="red" onClick={discard} ml={3}>
              Discard and switch
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialogOverlay>
    </AlertDialog>
  );

  return { request, markDirty, markClean, dialog };
}

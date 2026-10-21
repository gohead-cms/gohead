import { useCallback, useRef, useState } from "react";
import { Button } from "@chakra-ui/react";
import { DialogBody, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";

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
    <DialogRoot
      role="alertdialog"
      open={pending !== null}
      initialFocusEl={() => cancelRef.current}
      onOpenChange={(e) => {
        if (!e.open) setPending(null);
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle fontSize="lg" fontWeight="bold">
            Discard unsaved changes?
          </DialogTitle>
        </DialogHeader>
        <DialogBody>
          The changes made in this locale are not saved. Switching to {pending} loses them; stay to save them first.
        </DialogBody>
        <DialogFooter>
          <Button ref={cancelRef} variant="subtle" onClick={() => setPending(null)}>
            Keep editing
          </Button>
          <Button colorPalette="red" onClick={discard} ml={3}>
            Discard and switch
          </Button>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );

  return { request, markDirty, markClean, dialog };
}

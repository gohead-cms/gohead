import { useCallback } from "react";
import { useToast } from "@chakra-ui/react";
import { errorMessage } from "../utils/errors";

/** Toasts for the outcome of an action: success(msg) and failure(err). */
export function useNotify() {
  const toast = useToast();
  const success = useCallback(
    (title: string, description?: string) => toast({ title, description, status: "success", duration: 3000, isClosable: true }),
    [toast],
  );
  const failure = useCallback(
    (err: unknown, title = "Something went wrong") =>
      toast({ title, description: errorMessage(err), status: "error", duration: 7000, isClosable: true }),
    [toast],
  );
  return { success, failure };
}

import { useMemo } from "react";
import { toaster } from "@/components/ui/toaster";
import { errorMessage } from "../utils/errors";

/** Toasts for the outcome of an action: success, info, warning (msg) and failure(err). */
export function useNotify() {
  return useMemo(() => {
    const show = (type: "success" | "info" | "warning" | "error", duration: number, title: string, description?: string) =>
      toaster.create({ type, title, description, duration, closable: true });
    return {
      success: (title: string, description?: string) => show("success", 3000, title, description),
      info: (title: string, description?: string) => show("info", 4000, title, description),
      warning: (title: string, description?: string) => show("warning", 7000, title, description),
      failure: (err: unknown, title = "Something went wrong") => show("error", 7000, title, errorMessage(err)),
    };
  }, []);
}

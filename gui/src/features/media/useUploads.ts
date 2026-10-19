import { useCallback, useEffect, useRef, useState } from "react";
import { errorMessage } from "../../shared/utils/errors";
import { type MediaItem, type UploadTask, uploadFile } from "./media";

/**
 * Uploads files one request each, so that one refused file (type not
 * allowed, too large) does not fail the others (a request is atomic on the
 * server) and every file gets its own error.
 */
export function useUploads(onUploaded: (items: MediaItem[]) => void) {
  const [tasks, setTasks] = useState<UploadTask[]>([]);
  const seq = useRef(0);
  const onUploadedRef = useRef(onUploaded);
  useEffect(() => {
    onUploadedRef.current = onUploaded;
  });

  const patch = (id: string, change: Partial<UploadTask>) =>
    setTasks((all) => all.map((t) => (t.id === id ? { ...t, ...change } : t)));

  const upload = useCallback(async (files: File[]) => {
    if (files.length === 0) return;
    const batch = files.map((file) => ({ file, id: `u${++seq.current}` }));
    setTasks((all) => [
      ...all.filter((t) => t.status !== "done"),
      ...batch.map(({ file, id }) => ({ id, name: file.name, size: file.size, status: "pending" as const })),
    ]);
    const uploaded: MediaItem[] = [];
    for (const { file, id } of batch) {
      patch(id, { status: "uploading" });
      try {
        uploaded.push(await uploadFile(file));
        patch(id, { status: "done" });
      } catch (err) {
        patch(id, { status: "error", error: errorMessage(err, "Upload failed") });
      }
    }
    onUploadedRef.current(uploaded);
  }, []);

  const dismiss = useCallback((id: string) => setTasks((all) => all.filter((t) => t.id !== id)), []);
  const clearFinished = useCallback(
    () => setTasks((all) => all.filter((t) => t.status === "pending" || t.status === "uploading")),
    [],
  );
  const busy = tasks.some((t) => t.status === "pending" || t.status === "uploading");

  return { tasks, upload, dismiss, clearFinished, busy };
}

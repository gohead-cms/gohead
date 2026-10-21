import { useState } from "react";
import { Box, Button, Input, SimpleGrid, Spinner, Text, chakra } from "@chakra-ui/react";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { InputGroup } from "@/components/ui/input-group";
import { FiCheckCircle, FiSearch } from "react-icons/fi";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import { MediaThumb } from "../../media/components/MediaThumb";
import { type MediaItem, PAGE_SIZE, listPath } from "../../media/media";
import { mimeAllowed } from "./mediaRef";

/** The API's mime prefix filter for the patterns, when they share one type. */
function mimeFilter(accept?: string[]): string {
  if (!accept || accept.length === 0) return "";
  const types = new Set(accept.map((p) => p.split("/")[0]));
  return types.size === 1 ? `${[...types][0]}/` : "";
}

interface MediaLibraryPickerProps {
  multiple?: boolean;
  accept?: string[];
  /** IDs already in the field. */
  selected: number[];
  onClose: () => void;
  onPick: (files: MediaItem[]) => void;
}

/** Picks files already in the media library. */
export function MediaLibraryPicker({ multiple, accept, selected, onClose, onPick }: MediaLibraryPickerProps) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [picked, setPicked] = useState<MediaItem[]>([]);
  const path = listPath(page, mimeFilter(accept), search);
  const { data, loading, error } = useApiQuery(path, async () => {
    const json = await getJSON<MediaItem[]>(path, "Could not load the media library");
    return { items: json.data ?? [], total: Number((json.meta?.pagination as { total?: number } | undefined)?.total ?? 0) };
  });
  const pages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1;
  const isPicked = (m: MediaItem) => picked.some((p) => p.id === m.id);

  const choose = (m: MediaItem) => {
    if (!multiple) {
      onPick([m]);
      return;
    }
    setPicked((prev) => (isPicked(m) ? prev.filter((p) => p.id !== m.id) : [...prev, m]));
  };

  return (
    <DialogRoot
      open
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      size="xl"
      scrollBehavior="inside"
    >
      <DialogContent maxW="4xl">
        <DialogHeader>
          <DialogTitle>{multiple ? "Add files from the library" : "Choose a file from the library"}</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <InputGroup mb={4} w="full" startElement={<FiSearch />}>
            <Input
              autoFocus
              placeholder="Search by file name"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
            />
          </InputGroup>
          {error && (
            <Text color="red.500" fontSize="sm">
              {error}
            </Text>
          )}
          {!data && loading && <Spinner />}
          {data && data.items.length === 0 && (
            <Text color="gray.500" fontSize="sm">
              {search.trim() ? "No file matches." : "The library has no matching files yet."}
            </Text>
          )}
          <SimpleGrid minChildWidth="140px" gap={3}>
            {data?.items.map((m) => {
              const already = selected.includes(m.id);
              const allowed = mimeAllowed(m.mime, accept);
              const on = isPicked(m);
              return (
                <chakra.button
                  key={m.id}
                  type="button"
                  onClick={() => !already && allowed && choose(m)}
                  aria-disabled={already || !allowed}
                  title={!allowed ? `Not allowed in this field (${m.mime})` : already ? "Already in the field" : m.name}
                  aria-pressed={multiple ? on : undefined}
                  aria-label={`Choose ${m.name}${!allowed ? " (not allowed in this field)" : already ? " (already in the field)" : ""}`}
                  textAlign="left"
                  borderWidth="2px"
                  borderColor={on ? "purple.400" : "transparent"}
                  borderRadius="lg"
                  overflow="hidden"
                  position="relative"
                  opacity={already || !allowed ? 0.4 : 1}
                  cursor={already || !allowed ? "not-allowed" : "pointer"}
                  _hover={{ shadow: already || !allowed ? undefined : "md" }}
                >
                  <MediaThumb item={m} height="100px" />
                  {on && (
                    <Box position="absolute" top={1} right={1} color="purple.500" bg="white" borderRadius="full">
                      <FiCheckCircle size={20} />
                    </Box>
                  )}
                  <Text fontSize="xs" px={2} py={1} lineClamp={1}>
                    {m.name}
                  </Text>
                </chakra.button>
              );
            })}
          </SimpleGrid>
        </DialogBody>
        <DialogFooter gap={2}>
          <Button size="xs" variant="ghost" onClick={() => setPage((p) => p - 1)} disabled={page <= 1}>
            Previous
          </Button>
          <Text fontSize="sm" color="gray.500">
            {page} / {pages}
          </Text>
          <Button size="xs" variant="ghost" onClick={() => setPage((p) => p + 1)} disabled={page >= pages}>
            Next
          </Button>
          <Box flex="1" />
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          {multiple && (
            <Button colorPalette="purple" onClick={() => onPick(picked)} disabled={picked.length === 0}>
              Add {picked.length || ""}
            </Button>
          )}
        </DialogFooter>
        <DialogCloseTrigger />
      </DialogContent>
    </DialogRoot>
  );
}

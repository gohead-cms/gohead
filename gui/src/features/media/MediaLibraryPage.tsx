import { type ChangeEvent, type DragEvent, useEffect, useRef, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Center,
  Field,
  Flex,
  Icon,
  Input,
  NativeSelect,
  Spinner,
  Text,
  VisuallyHidden,
} from "@chakra-ui/react";
import { FiImage, FiLock, FiSearch, FiUpload, FiUploadCloud } from "react-icons/fi";
import { InputGroup } from "@/components/ui/input-group";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { useNotify } from "../../shared/hooks/useNotify";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { useMe } from "../auth/me";
import { MEDIA } from "../auth/permissions";
import { MediaDetailsDrawer } from "./components/MediaDetailsDrawer";
import { MediaGrid } from "./components/MediaGrid";
import { UploadQueue } from "./components/UploadQueue";
import { type MediaItem, type MediaPagination, TYPE_FILTERS, listPath } from "./media";
import { useUploads } from "./useUploads";

function hasFiles(e: DragEvent) {
  return Array.from(e.dataTransfer.types).includes("Files");
}

function Pager({ p, onPage }: { p: MediaPagination; onPage: (page: number) => void }) {
  if (p.pageCount <= 1) return null;
  return (
    <Flex justify="space-between" align="center" mt={6} gap={3} wrap="wrap">
      <Text fontSize="sm" color="gray.500">
        Page {p.page} of {p.pageCount} · {p.total} files
      </Text>
      <Flex gap={2}>
        <Button size="xs" variant="subtle" onClick={() => onPage(p.page - 1)} disabled={p.page <= 1}>
          Previous
        </Button>
        <Button size="xs" variant="subtle" onClick={() => onPage(p.page + 1)} disabled={p.page >= p.pageCount}>
          Next
        </Button>
      </Flex>
    </Flex>
  );
}

/** The media library: browse, search, upload and edit files. */
export function MediaLibraryPage() {
  const { can, loading: meLoading } = useMe();
  const canRead = can("read", MEDIA);
  const canUpload = can("create", MEDIA);
  const [searchInput, setSearchInput] = useState("");
  const [search, setSearch] = useState("");
  const [mime, setMime] = useState("");
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<number | null>(null);
  const [dragging, setDragging] = useState(false);
  const dragDepth = useRef(0);
  const fileInput = useRef<HTMLInputElement>(null);
  const notify = useNotify();

  // Search as the user types, once they pause.
  useEffect(() => {
    const t = setTimeout(() => {
      setSearch(searchInput);
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [searchInput]);

  const path = listPath(page, mime, search);
  const list = useApiQuery(canRead ? path : null, () => getJSON<MediaItem[]>(path, "Could not load the media library"));
  const items = list.data?.data ?? [];
  const pagination = list.data?.meta?.pagination as MediaPagination | undefined;
  const filtered = search.trim() !== "" || mime !== "";

  const uploads = useUploads((uploaded) => {
    if (uploaded.length === 0) return;
    notify.success(uploaded.length === 1 ? `Uploaded ${uploaded[0].name}` : `Uploaded ${uploaded.length} files`);
    setPage(1);
    list.refetch();
  });

  const pickFiles = () => fileInput.current?.click();
  const onPicked = (e: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files ?? []);
    e.target.value = "";
    void uploads.upload(files);
  };

  const dropHandlers = canUpload
    ? {
        onDragEnter: (e: DragEvent) => {
          if (!hasFiles(e)) return;
          e.preventDefault();
          dragDepth.current += 1;
          setDragging(true);
        },
        onDragOver: (e: DragEvent) => {
          if (!hasFiles(e)) return;
          e.preventDefault();
          e.dataTransfer.dropEffect = "copy";
        },
        onDragLeave: () => {
          dragDepth.current = Math.max(0, dragDepth.current - 1);
          if (dragDepth.current === 0) setDragging(false);
        },
        onDrop: (e: DragEvent) => {
          if (!hasFiles(e)) return;
          e.preventDefault();
          dragDepth.current = 0;
          setDragging(false);
          void uploads.upload(Array.from(e.dataTransfer.files));
        },
      }
    : {};

  const uploadButton = (
    <Button colorPalette="blue" onClick={pickFiles} loading={uploads.busy} loadingText="Uploading">
      <FiUpload />
      Upload files
    </Button>
  );

  if (meLoading) {
    return (
      <Page>
        <PageHeader title="Media library" />
        <Center py={16}>
          <Spinner />
        </Center>
      </Page>
    );
  }

  if (!canRead) {
    return (
      <Page>
        <PageHeader title="Media library" description="Images, video, audio and documents used by your content." />
        <EmptyState icon={FiLock} title="Your role has no access to the media library">
          Browsing and uploading files needs a role with media permissions. Ask an administrator for access; files used by content stay
          visible in that content.
        </EmptyState>
      </Page>
    );
  }

  let body;
  if (list.error) {
    body = (
      <Alert.Root status="error" borderRadius="md" alignItems="center">
        <Alert.Indicator />
        <Box flex="1">{list.error}</Box>
        <Button size="xs" variant="subtle" onClick={list.refetch}>
          Retry
        </Button>
      </Alert.Root>
    );
  } else if (!list.data) {
    body = (
      <Center py={16}>
        <Spinner />
      </Center>
    );
  } else if (items.length === 0 && page > 1) {
    body = (
      <EmptyState icon={FiImage} title="No files on this page" action={<Button size="xs" variant="subtle" onClick={() => setPage(1)}>Go to the first page</Button>} />
    );
  } else if (items.length === 0 && filtered) {
    body = (
      <EmptyState
        icon={FiSearch}
        title="No file matches"
        action={
          <Button
            size="xs"
            variant="subtle"
            onClick={() => {
              setSearchInput("");
              setSearch("");
              setMime("");
              setPage(1);
            }}
          >
            Clear filters
          </Button>
        }
      >
        Try another name or type.
      </EmptyState>
    );
  } else if (items.length === 0) {
    body = (
      <EmptyState icon={FiUploadCloud} title="No files yet" action={canUpload ? uploadButton : undefined}>
        {canUpload
          ? "Upload images, video, audio or documents, or drop files anywhere on this page. They can then be used in media fields."
          : "Files uploaded by others will be listed here."}
      </EmptyState>
    );
  } else {
    body = (
      <>
        <Box opacity={list.loading ? 0.6 : 1} transition="opacity 0.15s">
          <MediaGrid items={items} onOpen={setSelected} />
        </Box>
        {pagination && <Pager p={pagination} onPage={setPage} />}
      </>
    );
  }

  return (
    <Box position="relative" minH="100%" {...dropHandlers}>
      <Page>
        <PageHeader
          title="Media library"
          description={
            canUpload
              ? "Images, video, audio and documents used by your content. Drop files anywhere to upload them."
              : "Images, video, audio and documents used by your content."
          }
          actions={canUpload ? uploadButton : undefined}
        />
        <VisuallyHidden>
          <input ref={fileInput} type="file" multiple aria-label="Choose files to upload" onChange={onPicked} tabIndex={-1} />
        </VisuallyHidden>

        <Flex gap={3} mb={4} direction={{ base: "column", md: "row" }} align={{ base: "stretch", md: "flex-end" }}>
          <Field.Root flex="1" maxW={{ md: "sm" }}>
            <Field.Label fontSize="sm">Search</Field.Label>
            <InputGroup w="full" bg="white" startElement={<Icon as={FiSearch} color="gray.400" />}>
              <Input size="xs" type="search" placeholder="File name" value={searchInput} onChange={(e) => setSearchInput(e.target.value)} borderRadius="md" />
            </InputGroup>
          </Field.Root>
          <Field.Root w={{ base: "100%", md: "180px" }}>
            <Field.Label fontSize="sm">Type</Field.Label>
            <NativeSelect.Root size="xs">
              <NativeSelect.Field
                bg="white"
                borderRadius="md"
                value={mime}
                onChange={(e) => {
                  setMime(e.target.value);
                  setPage(1);
                }}
              >
                {TYPE_FILTERS.map((f) => (
                  <option key={f.value} value={f.value}>
                    {f.label}
                  </option>
                ))}
              </NativeSelect.Field>
              <NativeSelect.Indicator />
            </NativeSelect.Root>
          </Field.Root>
          {pagination && (
            <Text fontSize="sm" color="gray.500" ml={{ md: "auto" }} pb={1}>
              {pagination.total} {pagination.total === 1 ? "file" : "files"}
            </Text>
          )}
        </Flex>

        <UploadQueue tasks={uploads.tasks} busy={uploads.busy} onDismiss={uploads.dismiss} onClear={uploads.clearFinished} />
        {body}
      </Page>

      {dragging && (
        <Center
          position="fixed"
          inset={0}
          zIndex="overlay"
          bg="blackAlpha.500"
          pointerEvents="none"
          aria-hidden="true"
        >
          <Box bg="white" borderRadius="xl" borderWidth="2px" borderStyle="dashed" borderColor="blue.400" px={12} py={10} textAlign="center">
            <Icon as={FiUploadCloud} boxSize={10} color="blue.500" />
            <Text fontWeight="semibold" mt={2}>
              Drop files to upload them
            </Text>
          </Box>
        </Center>
      )}

      <MediaDetailsDrawer
        id={selected}
        canUpdate={can("update", MEDIA)}
        canDelete={can("delete", MEDIA)}
        onClose={() => setSelected(null)} onChanged={list.refetch} />
    </Box>
  );
}

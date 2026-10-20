import { useState } from "react";
import {
  Alert,
  AlertIcon,
  Badge,
  Box,
  Button,
  HStack,
  IconButton,
  Input,
  InputGroup,
  InputLeftElement,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Spinner,
  Text,
  VStack,
  useDisclosure,
} from "@chakra-ui/react";
import { FiCheckSquare, FiLink, FiSearch, FiSquare, FiX } from "react-icons/fi";
import { useMe } from "../../auth/me";
import { apiFetchWithAuth } from "../../../services/api";
import { useApiQuery } from "../../../shared/hooks/useApiQuery";
import type { Attribute } from "../../../shared/types";
import { useCollectionSchema } from "../../collections/hook/useCollectionSchema";
import { relationTarget } from "../../model/schema";
import { type RelatedEntry, entryLabel, labelField } from "./relationLabels";

type Ref = RelatedEntry | number | string;

const refID = (r: Ref) => (typeof r === "object" ? r.id : r);

function asList(value: unknown): Ref[] {
  if (value === null || value === undefined || value === "") return [];
  const list = Array.isArray(value) ? value : [value];
  return list.filter((v): v is Ref => typeof v === "number" || typeof v === "string" || (typeof v === "object" && v !== null && "id" in v));
}

/** GETs a page of a collection's entries, with drafts when the role may see them. */
async function listEntries(collection: string, query: string): Promise<{ items: RelatedEntry[]; total: number }> {
  const base = `/api/collections/${encodeURIComponent(collection)}?level=0&${query}`;
  let res = await apiFetchWithAuth(`${base}&status=all`);
  if (res.status === 403) res = await apiFetchWithAuth(base);
  const json = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(json.error?.message || `Could not list ${collection} (${res.status})`);
  return { items: Array.isArray(json.data) ? json.data : [], total: Number(json.meta?.pagination?.total ?? 0) };
}

interface RelationFieldProps {
  id: string;
  /** The attribute's name, for accessible names. */
  name: string;
  attr: Attribute;
  value: unknown;
  onChange: (value: unknown) => void;
}

/**
 * Picks the entries a relation points to. The form keeps populated entries
 * (for their labels) or bare IDs; saving sends the IDs.
 */
export function RelationField({ id, name, attr, value, onChange }: RelationFieldProps) {
  const target = relationTarget(attr.target);
  const many = attr.relation === "manyToMany";
  const { schema } = useCollectionSchema(target || null);
  const field = labelField(schema);
  const picker = useDisclosure();
  // The API leaves out relations to collections the role cannot read: such
  // a field looks empty, and saving the entry would clear it.
  const canRead = useMe().can("read", target);
  const selected = asList(value);

  // Labels of the entries the form only has the ID of.
  const bare = selected.filter((r) => typeof r !== "object").map(String);
  const labelsKey = bare.length > 0 && target ? `relation-labels:${target}:${bare.join(",")}` : null;
  const labels = useApiQuery(labelsKey, async () => {
    const out: Record<string, RelatedEntry | null> = {};
    await Promise.all(
      bare.map(async (ref) => {
        const res = await apiFetchWithAuth(`/api/collections/${encodeURIComponent(target)}/${encodeURIComponent(ref)}?level=0`);
        out[ref] = res.ok ? ((await res.json()).data as RelatedEntry) : null;
      }),
    );
    return out;
  });

  const resolve = (r: Ref): RelatedEntry | null => (typeof r === "object" ? r : (labels.data?.[String(r)] ?? null));
  const remove = (ref: Ref) => {
    const left = selected.filter((r) => String(refID(r)) !== String(refID(ref)));
    onChange(many ? left : null);
  };

  return (
    <VStack align="stretch" spacing={2} role="group" aria-labelledby={`${id}-label`}>
      {!canRead && (
        <Alert status="warning" borderRadius="md" fontSize="sm">
          <AlertIcon />
          Your role cannot read {target}: this relation is not shown, and saving the entry clears it.
        </Alert>
      )}
      {selected.map((r) => {
        const entry = resolve(r);
        const missing = typeof r !== "object" && labels.data && !entry;
        return (
          <HStack key={String(refID(r))} borderWidth="1px" borderRadius="md" px={3} py={2} spacing={3}>
            <Box as={FiLink} color="gray.400" flexShrink={0} />
            <Text flex="1" fontSize="sm" noOfLines={1}>
              {entry ? entryLabel(entry, field) : `#${refID(r)}`}
              {missing && (
                <Text as="span" color="gray.500">
                  {" "}
                  (not found, or not visible to you)
                </Text>
              )}
            </Text>
            {entry?.status === "draft" && <Badge colorScheme="yellow">draft</Badge>}
            <Text fontSize="xs" color="gray.400">
              #{refID(r)}
            </Text>
            <IconButton aria-label={`Remove ${entry ? entryLabel(entry, field) : `#${refID(r)}`} from ${name}`} icon={<FiX />} size="xs" variant="ghost" onClick={() => remove(r)} />
          </HStack>
        );
      })}
      <HStack>
        <Button
          size="sm"
          variant="outline"
          leftIcon={<FiLink />}
          onClick={picker.onOpen}
          isDisabled={!target || !canRead}
          aria-label={many ? `Add ${name} (${target})` : `${selected.length ? "Change" : "Choose"} ${name} (${target})`}
        >
          {many ? `Add ${target}` : selected.length ? `Change ${target}` : `Choose ${target}`}
        </Button>
        {labels.loading && <Spinner size="xs" />}
      </HStack>
      {picker.isOpen && (
        <RelationPicker
          target={target}
          title={many ? `Add ${name} (${target})` : `Choose ${name} (${target})`}
          many={many}
          field={field}
          selected={selected.map((r) => String(refID(r)))}
          onClose={picker.onClose}
          onPick={(entries) => {
            if (many) {
              const have = new Set(selected.map((r) => String(refID(r))));
              onChange([...selected, ...entries.filter((e) => !have.has(String(e.id)))]);
            } else {
              onChange(entries[0] ?? null);
            }
            picker.onClose();
          }}
        />
      )}
    </VStack>
  );
}

const PAGE_SIZE = 20;

interface RelationPickerProps {
  target: string;
  title: string;
  many: boolean;
  field: string | null;
  /** IDs already chosen. */
  selected: string[];
  onClose: () => void;
  onPick: (entries: RelatedEntry[]) => void;
}

/** Searches the target collection and returns the entries picked. */
function RelationPicker({ target, title, many, field, selected, onClose, onPick }: RelationPickerProps) {
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [picked, setPicked] = useState<RelatedEntry[]>([]);
  const term = search.trim();
  let filter = "";
  if (term) {
    if (field) filter = `&filters[${encodeURIComponent(field)}][$contains]=${encodeURIComponent(term)}`;
    // Without a text attribute, search is by ID (anything else matches none).
    else filter = `&filters[id]=${/^\d+$/.test(term) ? term : "0"}`;
  }
  const key = `relation-picker:${target}:${page}:${filter}`;
  const { data, loading, error } = useApiQuery(key, () => listEntries(target, `sort=id:desc&page=${page}&pageSize=${PAGE_SIZE}${filter}`));
  const pages = data ? Math.max(1, Math.ceil(data.total / PAGE_SIZE)) : 1;
  const isPicked = (e: RelatedEntry) => picked.some((p) => String(p.id) === String(e.id));

  const choose = (e: RelatedEntry) => {
    if (!many) {
      onPick([e]);
      return;
    }
    setPicked((prev) => (isPicked(e) ? prev.filter((p) => String(p.id) !== String(e.id)) : [...prev, e]));
  };

  return (
    <Modal isOpen onClose={onClose} size="lg" scrollBehavior="inside">
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>{title}</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <InputGroup mb={3}>
            <InputLeftElement pointerEvents="none">
              <FiSearch />
            </InputLeftElement>
            <Input
              autoFocus
              placeholder={field ? `Search by ${field}` : "Search by ID"}
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
              {term ? "No entry matches." : `No ${target} yet.`}
            </Text>
          )}
          <VStack align="stretch" spacing={1}>
            {data?.items.map((e) => {
              const already = selected.includes(String(e.id));
              return (
                <HStack
                  key={String(e.id)}
                  as="button"
                  type="button"
                  textAlign="left"
                  px={3}
                  py={2}
                  borderRadius="md"
                  borderWidth="1px"
                  borderColor={isPicked(e) ? "purple.400" : "transparent"}
                  bg={isPicked(e) ? "purple.50" : undefined}
                  _hover={{ bg: already ? undefined : "gray.50" }}
                  opacity={already ? 0.5 : 1}
                  disabled={already}
                  onClick={() => choose(e)}
                  aria-pressed={many ? isPicked(e) : undefined}
                >
                  {many && <Box as={already || isPicked(e) ? FiCheckSquare : FiSquare} color={isPicked(e) ? "purple.500" : "gray.400"} aria-hidden flexShrink={0} />}
                  <Text flex="1" fontSize="sm" noOfLines={1}>
                    {entryLabel(e, field)}
                  </Text>
                  {e.status === "draft" && <Badge colorScheme="yellow">draft</Badge>}
                  <Text fontSize="xs" color="gray.400">
                    #{String(e.id)}
                  </Text>
                </HStack>
              );
            })}
          </VStack>
        </ModalBody>
        <ModalFooter gap={2}>
          <Button size="sm" variant="ghost" onClick={() => setPage((p) => p - 1)} isDisabled={page <= 1}>
            Previous
          </Button>
          <Text fontSize="sm" color="gray.500">
            {page} / {pages}
          </Text>
          <Button size="sm" variant="ghost" onClick={() => setPage((p) => p + 1)} isDisabled={page >= pages}>
            Next
          </Button>
          <Box flex="1" />
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          {many && (
            <Button colorScheme="purple" onClick={() => onPick(picked)} isDisabled={picked.length === 0}>
              Add {picked.length || ""}
            </Button>
          )}
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}

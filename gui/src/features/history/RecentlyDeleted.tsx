import { useState } from "react";
import {
  Alert,
  AlertIcon,
  Button,
  Flex,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  Spinner,
  Table,
  TableContainer,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tooltip,
  Tr,
  useDisclosure,
} from "@chakra-ui/react";
import { FiRotateCcw, FiTrash } from "react-icons/fi";
import { useNotify } from "../../shared/hooks/useNotify";
import { formatShortTime, formatTime } from "../agents/components/runFormat";
import { DELETED_LIMIT, DELETES_SCANNED, type DeletedEntry, useDeletedEntries } from "./useHistory";
import { useMe } from "../auth/me";
import { type RestoreMeta, restoreBlockReason, restoreRevision, restoreToast } from "./history";
import { ActorTag } from "./RevisionBadges";

/**
 * A "Recently deleted" button for a collection: lists the entries deleted
 * and not brought back since, and restores one (the content it had when it
 * was deleted).
 */
export function RecentlyDeletedButton({
  collection,
  drafts,
  onRestored,
}: {
  collection: string;
  /** The collection is draft-and-publish: published entries need publish to come back. */
  drafts: boolean;
  onRestored: () => void;
}) {
  const { isOpen, onOpen, onClose } = useDisclosure();
  return (
    <>
      <Button bg="white" variant="outline" size="sm" leftIcon={<FiTrash />} onClick={onOpen}>
        Recently deleted
      </Button>
      <Modal isOpen={isOpen} onClose={onClose} size="3xl" scrollBehavior="inside">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Recently deleted</ModalHeader>
          <ModalCloseButton />
          <ModalBody pb={6}>{isOpen && <DeletedList collection={collection} drafts={drafts} onRestored={onRestored} />}</ModalBody>
        </ModalContent>
      </Modal>
    </>
  );
}

function DeletedList({ collection, drafts, onRestored }: { collection: string; drafts: boolean; onRestored: () => void }) {
  const notify = useNotify();
  const { can } = useMe();
  const rights = { create: can("create", collection), publish: can("publish", collection) };
  const { entries, loading, error, refetch } = useDeletedEntries(collection);
  const [restoring, setRestoring] = useState<number | null>(null);

  const restore = async (entry: DeletedEntry) => {
    const { entityId, version } = entry.revision;
    setRestoring(entityId);
    try {
      const res = await restoreRevision({ kind: "item", collection, id: entityId }, version);
      const toast = restoreToast(version, res.meta as RestoreMeta | undefined);
      notify[toast.status](toast.status === "success" ? `Entry #${entityId} brought back` : toast.title, toast.description);
      refetch();
      onRestored();
    } catch (err) {
      notify.failure(err, "Could not restore the entry");
    } finally {
      setRestoring(null);
    }
  };

  if (error) {
    return (
      <Alert status="error" borderRadius="md">
        <AlertIcon />
        {error}
      </Alert>
    );
  }
  if (loading && entries.length === 0) {
    return (
      <Flex justify="center" py={8}>
        <Spinner color="purple.500" />
      </Flex>
    );
  }
  if (entries.length === 0) {
    return (
      <>
        <Text color="gray.500">No deleted entries among the latest deletions in this collection.</Text>
        <ScanNote />
      </>
    );
  }
  return (
    <>
      <Text fontSize="sm" color="gray.500" mb={3}>
        Restoring brings an entry back with the content and publication state it had when it was deleted.
      </Text>
      <TableContainer borderWidth="1px" borderRadius="md">
        <Table size="sm" sx={{ "th, td": { px: 3 } }}>
          <Thead>
            <Tr>
              <Th>Entry</Th>
              <Th>Deleted by</Th>
              <Th>When</Th>
              <Th isNumeric />
            </Tr>
          </Thead>
          <Tbody>
            {entries.map((e) => (
              <Tr key={e.revision.entityId}>
                <Td maxW="240px">
                  <Text fontWeight="medium" noOfLines={1} title={e.title}>
                    {e.title ?? "Untitled"}
                  </Text>
                  <Text fontSize="xs" color="gray.500">
                    #{e.revision.entityId}
                  </Text>
                </Td>
                <Td>
                  <ActorTag revision={e.revision} />
                </Td>
                <Td whiteSpace="nowrap" title={formatTime(e.revision.createdAt)}>
                  {formatShortTime(e.revision.createdAt)}
                </Td>
                <Td isNumeric>
                  <RestoreButton
                    // The delete revision is the latest: restoring brings the entry back.
                    blocked={restoreBlockReason(e.revision, e.revision, rights, drafts)}
                    isLoading={restoring === e.revision.entityId}
                    isDisabled={restoring !== null && restoring !== e.revision.entityId}
                    onClick={() => restore(e)}
                  />
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </TableContainer>
      <ScanNote />
    </>
  );
}

function RestoreButton({
  blocked,
  isLoading,
  isDisabled,
  onClick,
}: {
  blocked: string | null;
  isLoading: boolean;
  isDisabled: boolean;
  onClick: () => void;
}) {
  const button = (
    <Button
      size="xs"
      colorScheme="purple"
      variant="outline"
      leftIcon={<FiRotateCcw />}
      isLoading={isLoading}
      isDisabled={isDisabled || !!blocked}
      onClick={onClick}
    >
      Restore
    </Button>
  );
  if (!blocked) return button;
  return (
    <Tooltip label={blocked} hasArrow>
      {/* A disabled button gets no hover events: the span shows the reason. */}
      <span tabIndex={0} aria-label={blocked}>
        {button}
      </span>
    </Tooltip>
  );
}

/** What the list covers: only the newest deletions are looked at. */
function ScanNote() {
  return (
    <Text fontSize="xs" color="gray.500" mt={3}>
      Looks at the latest {DELETES_SCANNED} deletions in this collection and lists up to {DELETED_LIMIT} entries. Older deletions are
      in Activity → Content changes, filtered by Deleted (admins), and each entry's history can still restore it.
    </Text>
  );
}

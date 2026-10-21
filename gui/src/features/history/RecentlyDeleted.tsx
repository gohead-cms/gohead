import { useState } from "react";
import { Alert, Button, Flex, Spinner, Table, Text, useDisclosure } from "@chakra-ui/react";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { Tooltip } from "@/components/ui/tooltip";
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
  const { open, onOpen, onClose } = useDisclosure();
  return (
    <>
      <Button bg="white" variant="outline" size="xs" onClick={onOpen}>
        <FiTrash />
        Recently deleted
      </Button>
      <DialogRoot
        open={open}
        onOpenChange={(e) => {
          if (!e.open) onClose();
        }}
        size="xl"
        scrollBehavior="inside"
      >
        <DialogContent maxW="3xl">
          <DialogHeader>
            <DialogTitle>Recently deleted</DialogTitle>
          </DialogHeader>
          <DialogBody pb={6}>{open && <DeletedList collection={collection} drafts={drafts} onRestored={onRestored} />}</DialogBody>
          <DialogCloseTrigger />
        </DialogContent>
      </DialogRoot>
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
      <Alert.Root role="alert" status="error" borderRadius="md">
        <Alert.Indicator />
        <Alert.Content>
          <Alert.Description>{error}</Alert.Description>
        </Alert.Content>
      </Alert.Root>
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
      <Table.ScrollArea whiteSpace="nowrap" borderWidth="1px" borderRadius="md">
        <Table.Root size="sm" css={{ "& th, & td": { px: 3 } }}>
          <Table.Header>
            <Table.Row>
              <Table.ColumnHeader>Entry</Table.ColumnHeader>
              <Table.ColumnHeader>Deleted by</Table.ColumnHeader>
              <Table.ColumnHeader>When</Table.ColumnHeader>
              <Table.ColumnHeader textAlign="end" />
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {entries.map((e) => (
              <Table.Row key={e.revision.entityId}>
                <Table.Cell maxW="240px">
                  <Text fontWeight="medium" lineClamp={1} title={e.title}>
                    {e.title ?? "Untitled"}
                  </Text>
                  <Text fontSize="xs" color="gray.500">
                    #{e.revision.entityId}
                  </Text>
                </Table.Cell>
                <Table.Cell>
                  <ActorTag revision={e.revision} />
                </Table.Cell>
                <Table.Cell whiteSpace="nowrap" title={formatTime(e.revision.createdAt)}>
                  {formatShortTime(e.revision.createdAt)}
                </Table.Cell>
                <Table.Cell textAlign="end">
                  <RestoreButton
                    // The delete revision is the latest: restoring brings the entry back.
                    blocked={restoreBlockReason(e.revision, e.revision, rights, drafts)}
                    loading={restoring === e.revision.entityId}
                    disabled={restoring !== null && restoring !== e.revision.entityId}
                    onClick={() => restore(e)}
                  />
                </Table.Cell>
              </Table.Row>
            ))}
          </Table.Body>
        </Table.Root>
      </Table.ScrollArea>
      <ScanNote />
    </>
  );
}

function RestoreButton({
  blocked,
  loading,
  disabled,
  onClick,
}: {
  blocked: string | null;
  loading: boolean;
  disabled: boolean;
  onClick: () => void;
}) {
  const button = (
    <Button
      size="2xs"
      colorPalette="purple"
      variant="outline"
      loading={loading}
      disabled={disabled || !!blocked}
      onClick={onClick}
    >
      <FiRotateCcw />
      Restore
    </Button>
  );
  if (!blocked) return button;
  return (
    <Tooltip content={blocked} showArrow>
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

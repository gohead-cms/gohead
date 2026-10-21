import { useState } from "react";
import { Alert, Box, Button, HStack, IconButton, NativeSelect, Spinner, Table, Text } from "@chakra-ui/react";
import { useColorModeValue } from "@/components/ui/color-mode";
import { FiRefreshCw } from "react-icons/fi";
import { formatShortTime, formatTime } from "../agents/components/runFormat";
import { type HistoryTarget, type Revision, revisionTarget } from "./history";
import { HistoryDrawer } from "./HistoryDrawer";
import { ActionBadge, ActorTag } from "./RevisionBadges";
import { type FeedFilters, useRevisionFeed } from "./useHistory";

interface ContentChangesProps {
  /** Filters always applied (e.g. one agent run). */
  fixed?: FeedFilters;
  /** Shows the actor kind and action filters. */
  filters?: boolean;
  pageSize?: number;
  /** Shown when nothing matches and no filter is set. */
  emptyText?: string;
}

/**
 * Content changes across entries, newest first (GET /api/revisions). Each
 * row opens the history of its entry.
 */
export function ContentChanges({ fixed, filters = false, pageSize = 25, emptyText = "No content changes yet." }: ContentChangesProps) {
  const [actorKind, setActorKind] = useState("");
  const [action, setAction] = useState("");
  const [page, setPage] = useState(1);
  const [open, setOpen] = useState<HistoryTarget | null>(null);
  const { revisions, pagination, loading, error, refetch } = useRevisionFeed({ ...fixed, actorKind, action }, page, pageSize);
  const hover = useColorModeValue("gray.50", "gray.700");
  const filtered = actorKind !== "" || action !== "";

  const openRow = (r: Revision) => setOpen(revisionTarget(r));

  return (
    <Box>
      <HStack mb={3} gap={3} wrap="wrap">
        {filters && (
          <>
            <NativeSelect.Root size="xs" maxW="170px">
              <NativeSelect.Field
                aria-label="Filter by actor"
                value={actorKind}
                onChange={(e) => {
                  setActorKind(e.target.value);
                  setPage(1);
                }}
              >
                <option value="">All actors</option>
                <option value="user">Users</option>
                <option value="token">API tokens</option>
                <option value="agent">Agents</option>
                <option value="system">System</option>
              </NativeSelect.Field>
              <NativeSelect.Indicator />
            </NativeSelect.Root>
            <NativeSelect.Root size="xs" maxW="170px">
              <NativeSelect.Field
                aria-label="Filter by action"
                value={action}
                onChange={(e) => {
                  setAction(e.target.value);
                  setPage(1);
                }}
              >
                <option value="">All actions</option>
                <option value="create">Created</option>
                <option value="update">Updated</option>
                <option value="publish">Published</option>
                <option value="unpublish">Unpublished</option>
                <option value="delete">Deleted</option>
                <option value="restore">Restored</option>
                <option value="baseline">Before history (baseline)</option>
              </NativeSelect.Field>
              <NativeSelect.Indicator />
            </NativeSelect.Root>
          </>
        )}
        <IconButton aria-label="Refresh content changes" size="xs" variant="subtle" onClick={refetch} loading={loading}>
          <FiRefreshCw />
        </IconButton>
        {pagination && (
          <Text fontSize="sm" color="gray.500">
            {pagination.total} change{pagination.total === 1 ? "" : "s"}
          </Text>
        )}
      </HStack>

      {error && (
        <Alert.Root role="alert" status="error" mb={3} borderRadius="md">
          <Alert.Indicator />
          <Alert.Content>
            <Alert.Description>{error}</Alert.Description>
          </Alert.Content>
        </Alert.Root>
      )}
      {loading && revisions.length === 0 && <Spinner size="sm" color="purple.500" />}
      {!loading && !error && revisions.length === 0 && (
        <Text fontSize="sm" color="gray.500">
          {filtered ? "No content changes match these filters." : emptyText}
        </Text>
      )}

      {revisions.length > 0 && (
        <Table.ScrollArea whiteSpace="nowrap" borderWidth="1px" borderRadius="md">
          <Table.Root size="sm" css={{ "& th, & td": { px: 3 } }}>
            <Table.Header>
              <Table.Row>
                <Table.ColumnHeader>When</Table.ColumnHeader>
                <Table.ColumnHeader>Entry</Table.ColumnHeader>
                <Table.ColumnHeader>Action</Table.ColumnHeader>
                <Table.ColumnHeader>Actor</Table.ColumnHeader>
                <Table.ColumnHeader textAlign="end">Version</Table.ColumnHeader>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {revisions.map((r) => {
                const entry = r.entityType === "singleton" ? r.typeName : `${r.typeName} #${r.entityId}`;
                return (
                  <Table.Row
                    key={r.id}
                    cursor="pointer"
                    _hover={{ bg: hover }}
                    tabIndex={0}
                    aria-label={`History of ${entry}`}
                    onClick={() => openRow(r)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") openRow(r);
                    }}
                  >
                    <Table.Cell whiteSpace="nowrap" title={formatTime(r.createdAt)}>
                      {formatShortTime(r.createdAt)}
                    </Table.Cell>
                    <Table.Cell maxW="200px" overflow="hidden" textOverflow="ellipsis" title={entry}>
                      <Text as="span" fontWeight="medium">
                        {r.typeName}
                      </Text>
                      {r.entityType === "singleton" ? (
                        <Text as="span" color="gray.500" fontSize="xs" ml={1}>
                          single type
                        </Text>
                      ) : (
                        <Text as="span" color="gray.500" ml={1}>
                          #{r.entityId}
                        </Text>
                      )}
                    </Table.Cell>
                    <Table.Cell>
                      <ActionBadge action={r.action} />
                    </Table.Cell>
                    <Table.Cell maxW="260px">
                      <ActorTag revision={r} links={false} />
                    </Table.Cell>
                    <Table.Cell textAlign="end">v{r.version}</Table.Cell>
                  </Table.Row>
                );
              })}
            </Table.Body>
          </Table.Root>
        </Table.ScrollArea>
      )}

      {pagination && pagination.pageCount > 1 && (
        <HStack mt={3} justify="flex-end">
          <Button size="xs" variant="subtle" onClick={() => setPage((p) => p - 1)} disabled={page <= 1}>
            Newer
          </Button>
          <Text fontSize="sm">
            {page} / {pagination.pageCount}
          </Text>
          <Button size="xs" variant="subtle" onClick={() => setPage((p) => p + 1)} disabled={page >= pagination.pageCount}>
            Older
          </Button>
        </HStack>
      )}

      <HistoryDrawer target={open} onClose={() => setOpen(null)} onRestored={refetch} />
    </Box>
  );
}

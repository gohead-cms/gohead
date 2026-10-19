import { useState } from "react";
import {
  Alert,
  AlertIcon,
  Box,
  Button,
  HStack,
  IconButton,
  Select,
  Spinner,
  Table,
  TableContainer,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
} from "@chakra-ui/react";
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
      <HStack mb={3} spacing={3} wrap="wrap">
        {filters && (
          <>
            <Select
              size="sm"
              maxW="170px"
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
            </Select>
            <Select
              size="sm"
              maxW="170px"
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
            </Select>
          </>
        )}
        <IconButton aria-label="Refresh content changes" size="sm" icon={<FiRefreshCw />} onClick={refetch} isLoading={loading} />
        {pagination && (
          <Text fontSize="sm" color="gray.500">
            {pagination.total} change{pagination.total === 1 ? "" : "s"}
          </Text>
        )}
      </HStack>

      {error && (
        <Alert status="error" mb={3} borderRadius="md">
          <AlertIcon />
          {error}
        </Alert>
      )}
      {loading && revisions.length === 0 && <Spinner size="sm" color="purple.500" />}
      {!loading && !error && revisions.length === 0 && (
        <Text fontSize="sm" color="gray.500">
          {filtered ? "No content changes match these filters." : emptyText}
        </Text>
      )}

      {revisions.length > 0 && (
        <TableContainer borderWidth="1px" borderRadius="md">
          <Table size="sm" sx={{ "th, td": { px: 3 } }}>
            <Thead>
              <Tr>
                <Th>When</Th>
                <Th>Entry</Th>
                <Th>Action</Th>
                <Th>Actor</Th>
                <Th isNumeric>Version</Th>
              </Tr>
            </Thead>
            <Tbody>
              {revisions.map((r) => {
                const entry = r.entityType === "singleton" ? r.typeName : `${r.typeName} #${r.entityId}`;
                return (
                  <Tr
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
                    <Td whiteSpace="nowrap" title={formatTime(r.createdAt)}>
                      {formatShortTime(r.createdAt)}
                    </Td>
                    <Td maxW="200px" overflow="hidden" textOverflow="ellipsis" title={entry}>
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
                    </Td>
                    <Td>
                      <ActionBadge action={r.action} />
                    </Td>
                    <Td maxW="260px">
                      <ActorTag revision={r} links={false} />
                    </Td>
                    <Td isNumeric>v{r.version}</Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        </TableContainer>
      )}

      {pagination && pagination.pageCount > 1 && (
        <HStack mt={3} justify="flex-end">
          <Button size="sm" onClick={() => setPage((p) => p - 1)} isDisabled={page <= 1}>
            Newer
          </Button>
          <Text fontSize="sm">
            {page} / {pagination.pageCount}
          </Text>
          <Button size="sm" onClick={() => setPage((p) => p + 1)} isDisabled={page >= pagination.pageCount}>
            Older
          </Button>
        </HStack>
      )}

      <HistoryDrawer target={open} onClose={() => setOpen(null)} onRestored={refetch} />
    </Box>
  );
}

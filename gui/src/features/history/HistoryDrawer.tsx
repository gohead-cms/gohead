import { useState } from "react";
import {
  Alert,
  AlertIcon,
  Box,
  Button,
  ButtonGroup,
  Code,
  Drawer,
  DrawerBody,
  DrawerCloseButton,
  DrawerContent,
  DrawerHeader,
  DrawerOverlay,
  Flex,
  Grid,
  Heading,
  HStack,
  Icon,
  Spinner,
  Text,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react";
import { FiClock, FiRotateCcw } from "react-icons/fi";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { formatShortTime, formatTime } from "../agents/components/runFormat";
import {
  type HistoryTarget,
  type RestoreMeta,
  type Revision,
  restoreRevision,
  restoredDescription,
  targetKey,
  targetLabel,
} from "./history";
import { ChangesTable } from "./ChangesTable";
import { ActionBadge, ActorTag, StatusBadge } from "./RevisionBadges";
import { useRevision, useRevisions } from "./useHistory";

export interface RestoredResult {
  version: number;
  /** The entry now, as the API returns items. */
  data: Record<string, unknown>;
  meta?: RestoreMeta;
}

interface HistoryDrawerProps {
  /** The entry whose history is shown; null closes the drawer. */
  target: HistoryTarget | null;
  onClose: () => void;
  /** Called after a version was restored, so the caller reloads the entry. */
  onRestored?: (result: RestoredResult) => void;
}

/**
 * The history of an entry: every version, newest first, the fields each one
 * changed, and restoring one. Needs the editor or admin role.
 */
export function HistoryDrawer({ target, onClose, onRestored }: HistoryDrawerProps) {
  return (
    <Drawer isOpen={target !== null} onClose={onClose} placement="right" size="xl">
      <DrawerOverlay />
      <DrawerContent maxW={{ base: "100vw", lg: "5xl" }}>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px">
          <HStack spacing={2}>
            <Icon as={FiClock} color="purple.500" />
            <Text>History</Text>
            {target && (
              <Text as="span" fontWeight="normal" color="gray.500" fontSize="md" noOfLines={1}>
                {targetLabel(target)}
              </Text>
            )}
          </HStack>
        </DrawerHeader>
        <DrawerBody p={0}>{target && <HistoryContent key={targetKey(target)} target={target} onRestored={onRestored} />}</DrawerBody>
      </DrawerContent>
    </Drawer>
  );
}

function HistoryContent({ target, onRestored }: { target: HistoryTarget; onRestored?: (result: RestoredResult) => void }) {
  const notify = useNotify();
  const { revisions, total, loading, error, refetch, hasMore, loadMore } = useRevisions(target);
  const [selected, setSelected] = useState<number | null>(null);
  const [compare, setCompare] = useState<"previous" | "current">("previous");
  const border = useColorModeValue("gray.200", "gray.600");
  const activeBg = useColorModeValue("purple.50", "purple.900");
  const hoverBg = useColorModeValue("gray.50", "gray.700");

  const latest = revisions[0];
  const current = revisions.find((r) => r.version === selected) ?? latest;

  const restore = async (version: number) => {
    try {
      const res = await restoreRevision(target, version);
      const meta = res.meta as RestoreMeta | undefined;
      notify.success(meta?.undeleted ? "Entry brought back" : "Version restored", restoredDescription(version, meta));
      setSelected(null);
      setCompare("previous");
      refetch();
      onRestored?.({ version, data: res.data, meta });
    } catch (err) {
      notify.failure(err, "Could not restore this version");
    }
  };

  if (error) {
    return (
      <Alert status="error" m={4} borderRadius="md" w="auto">
        <AlertIcon />
        {error}
      </Alert>
    );
  }
  if (loading && revisions.length === 0) {
    return (
      <Flex justify="center" py={12}>
        <Spinner color="purple.500" />
      </Flex>
    );
  }
  if (revisions.length === 0) {
    return (
      <Text color="gray.500" p={6}>
        No history yet: versions are recorded from the next change on.
      </Text>
    );
  }

  return (
    <Grid templateColumns={{ base: "1fr", lg: "300px minmax(0, 1fr)" }} h="full">
      <Box borderRightWidth={{ base: 0, lg: "1px" }} borderBottomWidth={{ base: "1px", lg: 0 }} borderColor={border} overflowY="auto" maxH={{ base: "40vh", lg: "none" }}>
        <Text fontSize="xs" color="gray.500" px={4} pt={3} pb={1}>
          {total} version{total === 1 ? "" : "s"}, newest first
        </Text>
        <VStack as="ul" listStyleType="none" spacing={0} align="stretch" aria-label="Versions">
          {revisions.map((r) => {
            const active = r.version === current?.version;
            return (
              <Box as="li" key={r.id}>
                <Box
                  as="button"
                  type="button"
                  w="full"
                  textAlign="left"
                  px={4}
                  py={2.5}
                  borderLeftWidth="3px"
                  borderLeftColor={active ? "purple.500" : "transparent"}
                  bg={active ? activeBg : "transparent"}
                  _hover={{ bg: active ? activeBg : hoverBg }}
                  aria-current={active ? "true" : undefined}
                  aria-label={`Version ${r.version}, ${r.action} by ${r.actorName || r.actorKind}`}
                  onClick={() => setSelected(r.version)}
                >
                  <RevisionSummary revision={r} isLatest={r.version === latest?.version} />
                </Box>
              </Box>
            );
          })}
        </VStack>
        {hasMore && (
          <Flex justify="center" py={3}>
            <Button size="xs" variant="outline" onClick={loadMore} isLoading={loading}>
              Load more ({total - revisions.length} older)
            </Button>
          </Flex>
        )}
      </Box>
      <Box overflowY="auto" p={{ base: 4, md: 6 }} minW={0}>
        {current && (
          <RevisionPanel
            key={current.version}
            target={target}
            revision={current}
            isLatest={current.version === latest?.version}
            compare={compare}
            onCompare={setCompare}
            onRestore={() => restore(current.version)}
          />
        )}
      </Box>
    </Grid>
  );
}

function RevisionSummary({ revision: r, isLatest }: { revision: Revision; isLatest: boolean }) {
  return (
    <VStack align="stretch" spacing={1}>
      <HStack spacing={2}>
        <Text fontWeight="bold" fontSize="sm">
          v{r.version}
        </Text>
        <ActionBadge action={r.action} />
        <StatusBadge status={r.status} />
        {isLatest && (
          <Text fontSize="xs" color="purple.600" fontWeight="semibold">
            current
          </Text>
        )}
        <Text fontSize="xs" color="gray.500" ml="auto" whiteSpace="nowrap" title={formatTime(r.createdAt)}>
          {formatShortTime(r.createdAt)}
        </Text>
      </HStack>
      <ActorTag revision={r} links={false} />
      {r.restoredFrom !== undefined && (
        <Text fontSize="xs" color="gray.500">
          restored from v{r.restoredFrom}
        </Text>
      )}
    </VStack>
  );
}

interface RevisionPanelProps {
  target: HistoryTarget;
  revision: Revision;
  isLatest: boolean;
  compare: "previous" | "current";
  onCompare: (c: "previous" | "current") => void;
  onRestore: () => Promise<void>;
}

/** One version: who made it, the fields it changed, and restoring it. */
function RevisionPanel({ target, revision: r, isLatest, compare, onCompare, onRestore }: RevisionPanelProps) {
  const { revision: detail, loading, error } = useRevision(target, r.version, compare);
  const [showData, setShowData] = useState(false);
  const codeBg = useColorModeValue("gray.50", "gray.800");
  // Restoring the current version changes nothing, unless it is a delete.
  const canRestore = !isLatest || r.action === "delete";

  const beforeLabel =
    compare === "current"
      ? `Current${detail?.comparedTo ? ` (v${detail.comparedTo})` : ""}`
      : detail?.comparedTo
        ? `v${detail.comparedTo}`
        : "Before";

  return (
    <VStack align="stretch" spacing={5}>
      <Flex justify="space-between" align="start" gap={3} wrap="wrap">
        <Box minW={0}>
          <HStack spacing={2} mb={1}>
            <Heading size="md">Version {r.version}</Heading>
            <ActionBadge action={r.action} />
            <StatusBadge status={r.status} />
          </HStack>
          <HStack spacing={2} wrap="wrap" color="gray.600" fontSize="sm">
            <ActorTag revision={r} />
            <Text>· {formatTime(r.createdAt)}</Text>
          </HStack>
          {r.restoredFrom !== undefined && (
            <Text fontSize="sm" color="gray.500" mt={1}>
              Restored from version {r.restoredFrom}.
            </Text>
          )}
        </Box>
        {canRestore ? (
          <ConfirmButton
            trigger={
              <Button size="sm" colorScheme="purple" leftIcon={<FiRotateCcw />}>
                Restore this version
              </Button>
            }
            title={`Restore version ${r.version}?`}
            body={
              r.action === "delete"
                ? "The entry was deleted in this version: restoring brings it back with the content it had. The restore is recorded as a new version."
                : `The entry's content becomes what it was in version ${r.version}. This is recorded as a new version, so nothing is lost.`
            }
            confirmLabel="Restore"
            confirmColorScheme="purple"
            onConfirm={onRestore}
          />
        ) : (
          <Text fontSize="sm" color="gray.500">
            This is the current version.
          </Text>
        )}
      </Flex>

      <Box>
        <Flex justify="space-between" align="center" mb={3} gap={3} wrap="wrap">
          <Text fontWeight="semibold">{compare === "current" ? "Differences from the current version" : "Changes in this version"}</Text>
          <ButtonGroup size="xs" isAttached variant="outline" role="group" aria-label="Compare with">
            <Button
              colorScheme={compare === "previous" ? "purple" : "gray"}
              variant={compare === "previous" ? "solid" : "outline"}
              aria-pressed={compare === "previous"}
              onClick={() => onCompare("previous")}
            >
              vs previous
            </Button>
            <Button
              colorScheme={compare === "current" ? "purple" : "gray"}
              variant={compare === "current" ? "solid" : "outline"}
              aria-pressed={compare === "current"}
              onClick={() => onCompare("current")}
            >
              vs current
            </Button>
          </ButtonGroup>
        </Flex>
        {loading && !detail && <Spinner size="sm" color="purple.500" />}
        {error && (
          <Alert status="error" borderRadius="md">
            <AlertIcon />
            {error}
          </Alert>
        )}
        {detail && !error && (
          <>
            {compare === "previous" && detail.comparedTo === null && (
              <Text fontSize="sm" color="gray.500" mb={2}>
                {r.version === 1 ? "The first version: every field it was created with." : "The previous version is no longer kept."}
              </Text>
            )}
            {compare === "current" && detail.changes.length > 0 && (
              <Text fontSize="sm" color="gray.500" mb={2}>
                What restoring this version would change.
              </Text>
            )}
            {detail.changes.length === 0 ? (
              <Text fontSize="sm" color="gray.500">
                {compare === "current" ? "Same content as the current version." : "No field changed in this version."}
              </Text>
            ) : (
              <ChangesTable changes={detail.changes} beforeLabel={beforeLabel} afterLabel={`v${r.version}`} />
            )}
          </>
        )}
      </Box>

      {detail?.data && (
        <Box>
          <Button size="xs" variant="link" colorScheme="purple" onClick={() => setShowData((v) => !v)}>
            {showData ? "Hide stored data" : "Show stored data"}
          </Button>
          {showData && (
            <Code display="block" whiteSpace="pre-wrap" wordBreak="break-word" p={3} mt={2} fontSize="xs" bg={codeBg} maxH="360px" overflowY="auto">
              {JSON.stringify(detail.data, null, 2)}
            </Code>
          )}
        </Box>
      )}
    </VStack>
  );
}

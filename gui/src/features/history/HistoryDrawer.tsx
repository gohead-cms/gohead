import { useState } from "react";
import { Alert, Box, Button, ButtonGroup, chakra, Code, Flex, Grid, Heading, HStack, Icon, Spinner, Text, VStack } from "@chakra-ui/react";
import { useColorModeValue } from "@/components/ui/color-mode";
import { DrawerBody, DrawerCloseTrigger, DrawerContent, DrawerHeader, DrawerRoot, DrawerTitle } from "@/components/ui/drawer";
import { Tooltip } from "@/components/ui/tooltip";
import { FiClock, FiRotateCcw } from "react-icons/fi";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { formatShortTime, formatTime } from "../agents/components/runFormat";
import { useMe } from "../auth/me";
import {
  type HistoryTarget,
  type RestoreMeta,
  type Revision,
  restoreBlockReason,
  restoreRevision,
  restoreToast,
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
    <DrawerRoot
      open={target !== null}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      placement="end"
      size="xl"
    >
      <DrawerContent maxW={{ base: "100vw", lg: "5xl" }}>
        <DrawerHeader borderBottomWidth="1px">
          <DrawerTitle display="flex" alignItems="center" gap={2} fontSize="xl" fontWeight="semibold" minW={0}>
            <Icon as={FiClock} color="purple.500" />
            <Text as="span">History</Text>
            {target && (
              <Text as="span" fontWeight="normal" color="gray.500" fontSize="md" lineClamp={1}>
                {targetLabel(target)}
              </Text>
            )}
          </DrawerTitle>
        </DrawerHeader>
        <DrawerBody p={0}>{target && <HistoryContent key={targetKey(target)} target={target} onRestored={onRestored} />}</DrawerBody>
        <DrawerCloseTrigger />
      </DrawerContent>
    </DrawerRoot>
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
      const toast = restoreToast(version, meta);
      notify[toast.status](toast.title, toast.description);
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
      <Alert.Root role="alert" status="error" m={4} borderRadius="md" w="auto">
        <Alert.Indicator />
        <Alert.Content>
          <Alert.Description>{error}</Alert.Description>
        </Alert.Content>
      </Alert.Root>
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
        <VStack as="ul" listStyleType="none" gap={0} align="stretch" aria-label="Versions">
          {revisions.map((r) => {
            const active = r.version === current?.version;
            return (
              <Box as="li" key={r.id}>
                <chakra.button
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
                </chakra.button>
              </Box>
            );
          })}
        </VStack>
        {hasMore && (
          <Flex justify="center" py={3}>
            <Button size="2xs" variant="outline" onClick={loadMore} loading={loading}>
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
            latest={latest}
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
    <VStack align="stretch" gap={1}>
      <HStack gap={2}>
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
  /** The entry's newest revision (a delete when the entry is deleted). */
  latest: Revision | undefined;
  compare: "previous" | "current";
  onCompare: (c: "previous" | "current") => void;
  onRestore: () => Promise<void>;
}

/** One version: who made it, the fields it changed, and restoring it. */
function RevisionPanel({ target, revision: r, isLatest, latest, compare, onCompare, onRestore }: RevisionPanelProps) {
  const { revision: detail, loading, error } = useRevision(target, r.version, compare);
  const [showData, setShowData] = useState(false);
  const codeBg = useColorModeValue("gray.50", "gray.800");
  const { can } = useMe();
  // Restoring the current version changes nothing, unless it is a delete.
  const canRestore = !isLatest || r.action === "delete";
  // Singletons have no draft state: history already needs update.
  const blocked =
    target.kind === "item"
      ? restoreBlockReason(
          r,
          latest,
          { create: can("create", target.collection), publish: can("publish", target.collection) },
          !!target.drafts,
        )
      : null;

  // A deleted entry has no current content: "current" is the delete
  // revision, which holds the content it had when it was deleted.
  const deleted = latest?.action === "delete";
  const beforeLabel =
    compare === "current"
      ? `${deleted ? "Deleted (last content)" : "Current"}${detail?.comparedTo ? ` (v${detail.comparedTo})` : ""}`
      : detail?.comparedTo
        ? `v${detail.comparedTo}`
        : "Before";

  return (
    <VStack align="stretch" gap={5}>
      <Flex justify="space-between" align="start" gap={3} wrap="wrap">
        <Box minW={0}>
          <HStack gap={2} mb={1}>
            <Heading size="xl" fontWeight="bold">
              Version {r.version}
            </Heading>
            <ActionBadge action={r.action} />
            <StatusBadge status={r.status} />
          </HStack>
          <HStack gap={2} wrap="wrap" color="gray.600" fontSize="sm">
            <ActorTag revision={r} />
            <Text>· {formatTime(r.createdAt)}</Text>
          </HStack>
          {r.restoredFrom !== undefined && (
            <Text fontSize="sm" color="gray.500" mt={1}>
              Restored from version {r.restoredFrom}.
            </Text>
          )}
        </Box>
        {canRestore && blocked ? (
          <Tooltip content={blocked} showArrow>
            {/* A disabled button gets no hover events: the span shows the reason. */}
            <span tabIndex={0} aria-label={blocked}>
              <Button size="xs" colorPalette="purple" disabled>
                <FiRotateCcw />
                Restore this version
              </Button>
            </span>
          </Tooltip>
        ) : canRestore ? (
          <ConfirmButton
            trigger={
              <Button size="xs" colorPalette="purple">
                <FiRotateCcw />
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
          <Text fontWeight="semibold">
            {compare === "current"
              ? deleted
                ? "Differences from the deleted content"
                : "Differences from the current version"
              : "Changes in this version"}
          </Text>
          <ButtonGroup size="2xs" attached variant="outline" role="group" aria-label="Compare with">
            <Button
              colorPalette={compare === "previous" ? "purple" : "gray"}
              variant={compare === "previous" ? "solid" : "outline"}
              aria-pressed={compare === "previous"}
              onClick={() => onCompare("previous")}
            >
              vs previous
            </Button>
            <Button
              colorPalette={compare === "current" ? "purple" : "gray"}
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
          <Alert.Root role="alert" status="error" borderRadius="md">
            <Alert.Indicator />
            <Alert.Content>
              <Alert.Description>{error}</Alert.Description>
            </Alert.Content>
          </Alert.Root>
        )}
        {detail && !error && (
          <>
            {compare === "previous" && detail.comparedTo === null && (
              <Text fontSize="sm" color="gray.500" mb={2}>
                {r.action === "baseline"
                  ? "Baseline: the content as it was when history started to be recorded."
                  : r.action === "create"
                    ? "The first version: every field it was created with."
                    : "The previous version is no longer kept."}
              </Text>
            )}
            {compare === "current" && deleted && (
              <Text fontSize="sm" color="gray.500" mb={2}>
                The entry is deleted: this compares with the content it had when it was deleted.
              </Text>
            )}
            {compare === "current" && !deleted && detail.changes.length > 0 && (
              <Text fontSize="sm" color="gray.500" mb={2}>
                What restoring this version would change.
              </Text>
            )}
            {detail.changes.length === 0 ? (
              <Text fontSize="sm" color="gray.500">
                {compare === "current"
                  ? deleted
                    ? "Same content as when the entry was deleted."
                    : "Same content as the current version."
                  : "No field changed in this version."}
              </Text>
            ) : (
              <ChangesTable changes={detail.changes} beforeLabel={beforeLabel} afterLabel={`v${r.version}`} />
            )}
          </>
        )}
      </Box>

      {detail?.data && (
        <Box>
          <Button
            size="2xs"
            variant="plain"
            colorPalette="purple"
            color="purple.500"
            px={0}
            h="auto"
            _hover={{ textDecoration: "underline" }}
            onClick={() => setShowData((v) => !v)}
          >
            {showData ? "Hide stored data" : "Show stored data"}
          </Button>
          {showData && (
            <Code display="block" whiteSpace="pre-wrap" wordBreak="break-word" p={3} mt={2} fontSize="xs" bg={codeBg} maxH="360px" overflowY="auto">
              {JSON.stringify(
                detail.translations && Object.keys(detail.translations).length > 0
                  ? { ...detail.data, translations: detail.translations }
                  : detail.data,
                null,
                2,
              )}
            </Code>
          )}
        </Box>
      )}
    </VStack>
  );
}

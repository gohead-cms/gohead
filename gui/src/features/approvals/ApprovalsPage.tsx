import { useEffect, useState } from "react";
import { Link as RouterLink, useSearchParams } from "react-router-dom";
import { Alert, Badge, Box, Button, HStack, Icon, IconButton, Input, Link, NativeSelect, Spinner, Table, Tabs, Tag, Text } from "@chakra-ui/react";
import { useColorModeValue } from "@/components/ui/color-mode";
import { FiAlertTriangle, FiCheckCircle, FiRefreshCw } from "react-icons/fi";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { formatShortTime, formatTime } from "../agents/components/runFormat";
import { useCollectionsList } from "../contrib/hooks";
import { ProposalBadges, ProposalDrawer } from "./ProposalDrawer";
import { entryLink, type Proposal, type ProposalFilters, proposalStatusColor, STATUS_TABS } from "./proposals";
import { useProposals } from "./useProposals";

/** The filters named in the URL: ?status=…&collection=…&agent=…&agentRun=… (pending by default). */
function filtersFromParams(params: URLSearchParams): ProposalFilters {
  const status = params.get("status");
  const run = Number(params.get("agentRun"));
  return {
    status: STATUS_TABS.some((t) => t.value === status) ? (status as ProposalFilters["status"]) : "pending",
    collection: params.get("collection") ?? "",
    agent: params.get("agent") ?? "",
    agentRun: Number.isInteger(run) && run > 0 ? run : undefined,
  };
}

/**
 * Content changes agents asked for (agents that need approval), for the
 * people who may make them: review, approve (as is or edited) or reject.
 */
export function ApprovalsPage() {
  const [params, setParams] = useSearchParams();
  const filters = filtersFromParams(params);
  const page = Math.max(1, Number(params.get("page")) || 1);
  const [open, setOpen] = useState<number | null>(null);
  // The agent filter is applied on Enter or when the field loses focus.
  const [agentInput, setAgentInput] = useState(filters.agent ?? "");
  // Follow the URL when it changes the agent filter (back/forward, a link).
  const [syncedAgent, setSyncedAgent] = useState(filters.agent ?? "");
  if ((filters.agent ?? "") !== syncedAgent) {
    setSyncedAgent(filters.agent ?? "");
    setAgentInput(filters.agent ?? "");
  }
  const { proposals, pagination, loading, error, refetch } = useProposals(filters, page);
  const { collections } = useCollectionsList();
  const hover = useColorModeValue("gray.50", "gray.700");
  const narrowed = Boolean(filters.collection || filters.agent || filters.agentRun);

  /** Sets filters (and goes back to the first page). */
  const update = (changes: Record<string, string | number | undefined>, keepPage = false) => {
    const next = new URLSearchParams(params);
    for (const [k, v] of Object.entries(changes)) {
      if (v === undefined || v === "") next.delete(k);
      else next.set(k, String(v));
    }
    if (!keepPage) next.delete("page");
    setParams(next, { replace: true });
  };
  const applyAgent = () => {
    if (agentInput.trim() !== (filters.agent ?? "")) update({ agent: agentInput.trim() });
  };
  const clearFilters = () => {
    setAgentInput("");
    update({ collection: undefined, agent: undefined, agentRun: undefined });
  };
  const showStatus = filters.status !== "pending";

  // Past the last page (its last proposal was just decided): go to the last one.
  const lastPage = pagination ? Math.max(1, pagination.pageCount) : null;
  useEffect(() => {
    if (lastPage === null || page <= lastPage) return;
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (lastPage > 1) next.set("page", String(lastPage));
        else next.delete("page");
        return next;
      },
      { replace: true },
    );
  }, [lastPage, page, setParams]);

  return (
    <Page>
      <PageHeader
        title="Approvals"
        description="Content changes proposed by agents that need an editor's approval. Nothing is written until you approve."
        actions={
          <IconButton aria-label="Refresh" size="xs" variant="subtle" onClick={refetch} loading={loading}>
            <FiRefreshCw />
          </IconButton>
        }
      />

      <Box bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 5 }}>
        <Tabs.Root
          value={filters.status}
          // v3 also calls this when the URL changes the tab: only a click resets the page.
          onValueChange={(e) => e.value !== filters.status && update({ status: e.value === "pending" ? undefined : e.value })}
          colorPalette="purple"
          size="sm"
          mb={4}
        >
          <Tabs.List overflowX="auto" overflowY="hidden">
            {STATUS_TABS.map((t) => (
              <Tabs.Trigger key={t.value} value={t.value} whiteSpace="nowrap" _selected={{ color: "purple.600" }}>
                {t.label}
              </Tabs.Trigger>
            ))}
          </Tabs.List>
        </Tabs.Root>

        <HStack mb={4} gap={3} wrap="wrap">
          <NativeSelect.Root size="xs" maxW="200px">
            <NativeSelect.Field aria-label="Filter by collection" value={filters.collection} onChange={(e) => update({ collection: e.target.value })}>
              <option value="">All collections</option>
              {filters.collection && !collections.some((c) => c.schema.collectionName === filters.collection) && (
                <option value={filters.collection}>{filters.collection}</option>
              )}
              {collections.map((c) => (
                <option key={c.schema.collectionName} value={c.schema.collectionName}>
                  {c.schema.info?.displayName || c.schema.collectionName}
                </option>
              ))}
            </NativeSelect.Field>
            <NativeSelect.Indicator />
          </NativeSelect.Root>
          <Input
            size="xs"
            maxW="200px"
            aria-label="Filter by agent"
            placeholder="Agent name"
            value={agentInput}
            onChange={(e) => setAgentInput(e.target.value)}
            onBlur={applyAgent}
            onKeyDown={(e) => {
              if (e.key === "Enter") applyAgent();
            }}
          />
          {filters.agentRun && (
            <Tag.Root size="md" variant="subtle" colorPalette="purple" borderRadius="full">
              <Tag.Label>Run #{filters.agentRun}</Tag.Label>
              <Tag.EndElement>
                <Tag.CloseTrigger aria-label="Show every run" onClick={() => update({ agentRun: undefined })} />
              </Tag.EndElement>
            </Tag.Root>
          )}
          {pagination && (
            <Text fontSize="sm" color="gray.500" ml="auto">
              {pagination.total} proposal{pagination.total === 1 ? "" : "s"}
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
        {loading && proposals.length === 0 && <Spinner />}
        {!loading && !error && proposals.length === 0 &&
          (narrowed ? (
            <Text fontSize="sm" color="gray.500">
              No proposals match these filters.{" "}
              <Button variant="plain" size="xs" color="gray.500" px={0} h="auto" _hover={{ textDecoration: "underline" }} onClick={clearFilters}>
                Clear filters
              </Button>
            </Text>
          ) : (
            <EmptyState icon={FiCheckCircle} title={filters.status === "pending" ? "Nothing to review" : `No ${filters.status === "all" ? "" : `${filters.status} `}proposals`}>
              Agents that need approval propose their content changes here instead of writing them. Turn it on with “Require approval” in an
              agent's settings.
            </EmptyState>
          ))}

        {proposals.length > 0 && (
          <Table.ScrollArea whiteSpace="nowrap" borderWidth="1px" borderRadius="md">
            <Table.Root size="sm" css={{ "& th, & td": { px: 3 } }}>
              <Table.Header>
                <Table.Row>
                  <Table.ColumnHeader>Proposed</Table.ColumnHeader>
                  <Table.ColumnHeader>Agent</Table.ColumnHeader>
                  <Table.ColumnHeader>Change</Table.ColumnHeader>
                  <Table.ColumnHeader>Entry</Table.ColumnHeader>
                  <Table.ColumnHeader>Reason</Table.ColumnHeader>
                  {showStatus && <Table.ColumnHeader>Status</Table.ColumnHeader>}
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {proposals.map((p) => (
                  <ProposalRow key={p.id} proposal={p} stale={Boolean(p.stale)} showStatus={showStatus} hover={hover} onOpen={() => setOpen(p.id)} />
                ))}
              </Table.Body>
            </Table.Root>
          </Table.ScrollArea>
        )}

        {pagination && pagination.pageCount > 1 && (
          <HStack mt={3} justify="flex-end">
            <Button size="xs" variant="subtle" onClick={() => update({ page: page - 1 }, true)} disabled={page <= 1}>
              Newer
            </Button>
            <Text fontSize="sm">
              {page} / {pagination.pageCount}
            </Text>
            <Button size="xs" variant="subtle" onClick={() => update({ page: page + 1 }, true)} disabled={page >= pagination.pageCount}>
              Older
            </Button>
          </HStack>
        )}
      </Box>

      <ProposalDrawer proposalId={open} onClose={() => setOpen(null)} />
    </Page>
  );
}

function ProposalRow({
  proposal: p,
  stale,
  showStatus,
  hover,
  onOpen,
}: {
  proposal: Proposal;
  stale: boolean;
  showStatus: boolean;
  hover: string;
  onOpen: () => void;
}) {
  return (
    // The whole row opens the proposal with the mouse; the button in its
    // first cell does it from the keyboard.
    <Table.Row cursor="pointer" _hover={{ bg: hover }} onClick={onOpen}>
      <Table.Cell whiteSpace="nowrap" title={formatTime(p.createdAt)}>
        <Button
          variant="plain"
          size="xs"
          px={0}
          h="auto"
          fontWeight="normal"
          color="inherit"
          _hover={{ textDecoration: "underline" }}
          aria-label={`Open proposal ${p.id} by ${p.agentName}`}
          onClick={(e) => {
            e.stopPropagation();
            onOpen();
          }}
        >
          {formatShortTime(p.createdAt)}
        </Button>
      </Table.Cell>
      <Table.Cell maxW="160px" overflow="hidden" textOverflow="ellipsis" fontWeight="medium" title={p.agentName}>
        {p.agentName}
      </Table.Cell>
      <Table.Cell>
        <ProposalBadges proposal={p} />
      </Table.Cell>
      <Table.Cell maxW="220px" overflow="hidden" textOverflow="ellipsis">
        <HStack gap={1.5}>
          <Text as="span" fontWeight="medium">
            {p.collection}
          </Text>
          {p.itemId !== undefined ? (
            <Link asChild color="purple.600" onClick={(e) => e.stopPropagation()} title="Open the entry">
              <RouterLink to={entryLink(p.collection, p.itemId)}>#{p.itemId}</RouterLink>
            </Link>
          ) : (
            <Text as="span" color="gray.500" fontSize="xs">
              new entry
            </Text>
          )}
          {stale && (
            <Badge colorPalette="orange" variant="subtle" display="inline-flex" alignItems="center" gap={1} title="The entry changed since the agent proposed this">
              <Icon as={FiAlertTriangle} />
              changed since
            </Badge>
          )}
        </HStack>
      </Table.Cell>
      <Table.Cell maxW="320px" overflow="hidden" textOverflow="ellipsis" title={p.reason || undefined} color={p.reason ? undefined : "gray.400"}>
        {p.reason || "–"}
      </Table.Cell>
      {showStatus && (
        <Table.Cell>
          <Badge colorPalette={proposalStatusColor[p.status] ?? "gray"} title={p.error || p.reviewNote || undefined}>
            {p.status}
          </Badge>
        </Table.Cell>
      )}
    </Table.Row>
  );
}

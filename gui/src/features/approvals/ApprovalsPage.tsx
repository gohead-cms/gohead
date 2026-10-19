import { useState } from "react";
import { Link as RouterLink, useSearchParams } from "react-router-dom";
import {
  Alert,
  AlertIcon,
  Badge,
  Box,
  Button,
  HStack,
  Icon,
  IconButton,
  Input,
  Link,
  Select,
  Spinner,
  Tab,
  TabList,
  Table,
  TableContainer,
  Tabs,
  Tag,
  TagCloseButton,
  TagLabel,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useColorModeValue,
} from "@chakra-ui/react";
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
  const { proposals, stale, pagination, loading, error, refetch } = useProposals(filters, page);
  const { collections } = useCollectionsList();
  const hover = useColorModeValue("gray.50", "gray.700");
  const tabIndex = Math.max(0, STATUS_TABS.findIndex((t) => t.value === filters.status));
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

  return (
    <Page>
      <PageHeader
        title="Approvals"
        description="Content changes proposed by agents that need an editor's approval. Nothing is written until you approve."
        actions={<IconButton aria-label="Refresh" icon={<FiRefreshCw />} size="sm" onClick={refetch} isLoading={loading} />}
      />

      <Box bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 5 }}>
        <Tabs
          index={tabIndex}
          onChange={(i) => update({ status: STATUS_TABS[i].value === "pending" ? undefined : STATUS_TABS[i].value })}
          colorScheme="purple"
          size="sm"
          mb={4}
        >
          <TabList overflowX="auto" overflowY="hidden">
            {STATUS_TABS.map((t) => (
              <Tab key={t.value} whiteSpace="nowrap">
                {t.label}
              </Tab>
            ))}
          </TabList>
        </Tabs>

        <HStack mb={4} spacing={3} wrap="wrap">
          <Select
            size="sm"
            maxW="200px"
            aria-label="Filter by collection"
            value={filters.collection}
            onChange={(e) => update({ collection: e.target.value })}
          >
            <option value="">All collections</option>
            {filters.collection && !collections.some((c) => c.schema.collectionName === filters.collection) && (
              <option value={filters.collection}>{filters.collection}</option>
            )}
            {collections.map((c) => (
              <option key={c.schema.collectionName} value={c.schema.collectionName}>
                {c.schema.info?.displayName || c.schema.collectionName}
              </option>
            ))}
          </Select>
          <Input
            size="sm"
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
            <Tag size="md" colorScheme="purple" borderRadius="full">
              <TagLabel>Run #{filters.agentRun}</TagLabel>
              <TagCloseButton aria-label="Show every run" onClick={() => update({ agentRun: undefined })} />
            </Tag>
          )}
          {pagination && (
            <Text fontSize="sm" color="gray.500" ml="auto">
              {pagination.total} proposal{pagination.total === 1 ? "" : "s"}
            </Text>
          )}
        </HStack>

        {error && (
          <Alert status="error" mb={3} borderRadius="md">
            <AlertIcon />
            {error}
          </Alert>
        )}
        {loading && proposals.length === 0 && <Spinner />}
        {!loading && !error && proposals.length === 0 &&
          (narrowed ? (
            <Text fontSize="sm" color="gray.500">
              No proposals match these filters.{" "}
              <Button variant="link" size="sm" onClick={clearFilters}>
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
          <TableContainer borderWidth="1px" borderRadius="md">
            <Table size="sm" sx={{ "th, td": { px: 3 } }}>
              <Thead>
                <Tr>
                  <Th>Proposed</Th>
                  <Th>Agent</Th>
                  <Th>Change</Th>
                  <Th>Entry</Th>
                  <Th>Reason</Th>
                  {showStatus && <Th>Status</Th>}
                </Tr>
              </Thead>
              <Tbody>
                {proposals.map((p) => (
                  <ProposalRow key={p.id} proposal={p} stale={stale.has(p.id)} showStatus={showStatus} hover={hover} onOpen={() => setOpen(p.id)} />
                ))}
              </Tbody>
            </Table>
          </TableContainer>
        )}

        {pagination && pagination.pageCount > 1 && (
          <HStack mt={3} justify="flex-end">
            <Button size="sm" onClick={() => update({ page: page - 1 }, true)} isDisabled={page <= 1}>
              Newer
            </Button>
            <Text fontSize="sm">
              {page} / {pagination.pageCount}
            </Text>
            <Button size="sm" onClick={() => update({ page: page + 1 }, true)} isDisabled={page >= pagination.pageCount}>
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
    <Tr
      cursor="pointer"
      _hover={{ bg: hover }}
      tabIndex={0}
      aria-label={`Proposal ${p.id} by ${p.agentName}`}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === "Enter") onOpen();
      }}
    >
      <Td whiteSpace="nowrap" title={formatTime(p.createdAt)}>
        {formatShortTime(p.createdAt)}
      </Td>
      <Td maxW="160px" overflow="hidden" textOverflow="ellipsis" fontWeight="medium" title={p.agentName}>
        {p.agentName}
      </Td>
      <Td>
        <ProposalBadges proposal={p} />
      </Td>
      <Td maxW="220px" overflow="hidden" textOverflow="ellipsis">
        <HStack spacing={1.5}>
          <Text as="span" fontWeight="medium">
            {p.collection}
          </Text>
          {p.itemId !== undefined ? (
            <Link
              as={RouterLink}
              to={entryLink(p.collection, p.itemId)}
              color="purple.600"
              onClick={(e) => e.stopPropagation()}
              title="Open the entry"
            >
              #{p.itemId}
            </Link>
          ) : (
            <Text as="span" color="gray.500" fontSize="xs">
              new entry
            </Text>
          )}
          {stale && (
            <Badge colorScheme="orange" variant="subtle" display="inline-flex" alignItems="center" gap={1} title="The entry changed since the agent proposed this">
              <Icon as={FiAlertTriangle} />
              changed since
            </Badge>
          )}
        </HStack>
      </Td>
      <Td maxW="320px" overflow="hidden" textOverflow="ellipsis" title={p.reason || undefined} color={p.reason ? undefined : "gray.400"}>
        {p.reason || "–"}
      </Td>
      {showStatus && (
        <Td>
          <Badge colorScheme={proposalStatusColor[p.status] ?? "gray"} title={p.error || p.reviewNote || undefined}>
            {p.status}
          </Badge>
        </Td>
      )}
    </Tr>
  );
}

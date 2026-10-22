import { useState } from "react";
import { Alert, Badge, Box, Button, Code, Collapsible, HStack, IconButton, Skeleton, Stack, Text, VStack } from "@chakra-ui/react";
import { FiChevronDown, FiChevronRight, FiEdit2, FiLink2, FiPlus, FiRefreshCw, FiTrash2 } from "react-icons/fi";
import { Tooltip } from "@/components/ui/tooltip";
import { api } from "../../services/api";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { optTime } from "../settings/components/settingsFormat";
import { ConnectionFormDialog } from "./ConnectionFormDialog";
import type { Connection, ConnectionPreset, ConnectionTool } from "./types";

/** Editing state: closed, connecting (null) or editing a connection. */
type Editing = { connection: Connection | null; n: number } | undefined;

function ToolRow({ conn, tool }: { conn: Connection; tool: ConnectionTool }) {
  return (
    <Box py={2} borderTopWidth="1px" borderColor="gray.100">
      <HStack gap={2} wrap="wrap">
        <Text fontSize="sm" fontWeight="medium">
          {tool.title || tool.name}
        </Text>
        <Code fontSize="xs" bg="transparent" px={0} color="gray.500">
          {conn.name}.{tool.name}
        </Code>
        {tool.readOnly ? (
          <Badge size="sm" colorPalette="green" variant="subtle">
            Reads
          </Badge>
        ) : (
          <Badge size="sm" colorPalette="orange" variant="subtle" title="Calls wait for an admin's approval unless an agent says otherwise">
            Writes · approval
          </Badge>
        )}
      </HStack>
      {tool.description && (
        <Text fontSize="xs" color="gray.600" lineClamp={2} title={tool.description}>
          {tool.description}
        </Text>
      )}
    </Box>
  );
}

function ConnectionCard({
  conn,
  refreshing,
  onRefresh,
  onEdit,
  onDelete,
}: {
  conn: Connection;
  refreshing: boolean;
  onRefresh: () => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const [open, setOpen] = useState(false);
  const writes = conn.tools.filter((t) => !t.readOnly).length;
  return (
    <Box bg="white" borderWidth="1px" borderRadius="lg" p={4} data-testid={`connection-${conn.name}`}>
      <HStack justify="space-between" align="start" gap={3}>
        <Box minW={0}>
          <HStack gap={2} wrap="wrap">
            <Text fontWeight="semibold">{conn.label}</Text>
            <Code fontSize="xs">{conn.name}</Code>
            {conn.lastError ? (
              <Badge colorPalette="red">Not connected</Badge>
            ) : (
              <Badge colorPalette="green">
                {conn.tools.length} tool{conn.tools.length === 1 ? "" : "s"}
              </Badge>
            )}
            {conn.authType !== "none" && !conn.hasToken && <Badge colorPalette="orange">No token</Badge>}
          </HStack>
          <Text fontSize="xs" color="gray.500" wordBreak="break-all" lineClamp={1} title={conn.url}>
            {conn.url}
          </Text>
          {conn.toolsUpdatedAt && (
            <Text fontSize="xs" color="gray.500">
              Tools listed {optTime(conn.toolsUpdatedAt)}
              {writes > 0 && ` · ${writes} may change things`}
            </Text>
          )}
        </Box>
        <HStack gap={0} flexShrink={0}>
          <Tooltip content="Refresh the tools" showArrow>
            <IconButton aria-label={`Refresh ${conn.label}`} size="xs" variant="ghost" loading={refreshing} onClick={onRefresh}>
              <FiRefreshCw />
            </IconButton>
          </Tooltip>
          <Tooltip content="Edit" showArrow>
            <IconButton aria-label={`Edit ${conn.label}`} size="xs" variant="ghost" onClick={onEdit}>
              <FiEdit2 />
            </IconButton>
          </Tooltip>
          <ConfirmButton
            title={`Disconnect ${conn.label}?`}
            body="Its token is deleted, agents that use its tools get an error when they call them, and calls waiting for approval are rejected."
            confirmLabel="Disconnect"
            onConfirm={onDelete}
            trigger={
              <IconButton aria-label={`Delete ${conn.label}`} size="xs" variant="ghost" colorPalette="red">
                <FiTrash2 />
              </IconButton>
            }
          />
        </HStack>
      </HStack>
      {conn.lastError && (
        <Alert.Root role="alert" status="error" borderRadius="md" mt={3} fontSize="sm">
          <Alert.Indicator />
          <Alert.Content>
            <Alert.Description wordBreak="break-word">{conn.lastError}</Alert.Description>
          </Alert.Content>
        </Alert.Root>
      )}
      {conn.tools.length > 0 && (
        <Collapsible.Root open={open} onOpenChange={(e) => setOpen(e.open)} mt={2}>
          <Collapsible.Trigger asChild>
            <Button size="xs" variant="plain" color="gray.600" px={0} h="auto" _hover={{ textDecoration: "underline" }}>
              {open ? <FiChevronDown /> : <FiChevronRight />}
              {open ? "Hide the tools" : "Show the tools"}
            </Button>
          </Collapsible.Trigger>
          <Collapsible.Content>
            <VStack align="stretch" gap={0} mt={2}>
              {conn.tools.map((t) => (
                <ToolRow key={t.name} conn={conn} tool={t} />
              ))}
            </VStack>
          </Collapsible.Content>
        </Collapsible.Root>
      )}
    </Box>
  );
}

export function ConnectionsPage() {
  const notify = useNotify();
  const [editing, setEditing] = useState<Editing>(undefined);
  const [refreshing, setRefreshing] = useState<string | null>(null);
  const { data, loading, error, refetch } = useApiQuery("/admin/connections", async () => {
    const [list, presets] = await Promise.all([
      getJSON<Connection[]>("/admin/connections", "Failed to load the connected apps"),
      getJSON<ConnectionPreset[]>("/admin/connection-presets", "Failed to load the app catalog"),
    ]);
    return {
      connections: Array.isArray(list.data) ? list.data : [],
      presets: Array.isArray(presets.data) ? presets.data : [],
      encrypted: presets.meta?.encrypted !== false,
    };
  });
  const connections = data?.connections ?? [];

  const openForm = (connection: Connection | null) => setEditing((e) => ({ connection, n: (e?.n ?? 0) + 1 }));

  const refresh = async (c: Connection) => {
    setRefreshing(c.name);
    try {
      const res = await api<Connection>("POST", `/admin/connections/${c.name}/refresh`);
      if (typeof res.meta?.warning === "string") notify.failure(new Error(res.meta.warning), `${c.label}: could not list the tools`);
      else notify.success(`${c.label}: ${res.data.tools.length} tools`);
      refetch();
    } catch (err) {
      notify.failure(err, "Could not refresh the tools");
    } finally {
      setRefreshing(null);
    }
  };

  const remove = async (c: Connection) => {
    try {
      await api("DELETE", `/admin/connections/${c.name}`);
      notify.success(`${c.label} disconnected`);
      refetch();
    } catch (err) {
      notify.failure(err, "Could not delete the connection");
    }
  };

  const newButton = (
    <Button colorPalette="blue" onClick={() => openForm(null)} disabled={!data}>
      <FiPlus />
      Connect an app
    </Button>
  );

  return (
    <Page maxW="5xl">
      <PageHeader
        title="Connected apps"
        description="Give agents tools from other services: GitHub, Gmail, Google Drive or Slack through Zapier, or any MCP server. Agents add them under Tools → Connected apps; calls that may change something wait on the Approvals page."
        actions={newButton}
      />

      {error && (
        <Alert.Root role="alert" status="error" borderRadius="md" mb={4} alignItems="center">
          <Alert.Indicator />
          <Alert.Content flex="1">
            <Alert.Description>{error}</Alert.Description>
          </Alert.Content>
          <Button size="xs" variant="subtle" onClick={refetch}>
            Retry
          </Button>
        </Alert.Root>
      )}

      {loading && !data ? (
        <Stack bg="white" borderWidth="1px" borderRadius="lg" p={4}>
          {[0, 1].map((i) => (
            <Skeleton key={i} h="60px" />
          ))}
        </Stack>
      ) : data && connections.length === 0 ? (
        <EmptyState icon={FiLink2} title="No connected apps" action={newButton}>
          Connect GitHub, Zapier (Gmail, Drive, Sheets, Slack…) or any MCP server, and agents can use their tools.
        </EmptyState>
      ) : data ? (
        <VStack align="stretch" gap={3}>
          {connections.map((c) => (
            <ConnectionCard
              key={c.name}
              conn={c}
              refreshing={refreshing === c.name}
              onRefresh={() => refresh(c)}
              onEdit={() => openForm(c)}
              onDelete={() => remove(c)}
            />
          ))}
        </VStack>
      ) : null}

      {editing && data && (
        <ConnectionFormDialog
          key={editing.n}
          connection={editing.connection}
          presets={data.presets}
          encrypted={data.encrypted}
          onClose={() => setEditing(undefined)}
          onSaved={(saved, created, warning) => {
            setEditing(undefined);
            if (warning) notify.failure(new Error(warning), `${saved.label} saved, but not connected`);
            else notify.success(created ? `${saved.label} connected: ${saved.tools.length} tools` : `${saved.label} saved`);
            refetch();
          }}
        />
      )}
    </Page>
  );
}

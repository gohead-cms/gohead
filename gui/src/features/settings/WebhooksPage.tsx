import { useState } from "react";
import { Alert, Badge, Box, Button, Code, HStack, IconButton, Skeleton, Stack, Table, Text, Wrap } from "@chakra-ui/react";
import { Tooltip } from "@/components/ui/tooltip";
import { FiEdit2, FiList, FiPlus, FiSend, FiTrash2, FiZap } from "react-icons/fi";
import { api } from "../../services/api";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { formatDuration } from "../agents/components/runFormat";
import { DeliveriesDrawer } from "./components/DeliveriesDrawer";
import { SecretDialog } from "./components/SecretDialog";
import { WebhookFormDrawer } from "./components/WebhookFormDrawer";
import { deliveryFailed, optTime } from "./components/settingsFormat";
import type { Webhook, WebhookTestResult } from "./components/types";

/** Editing state: closed, creating (null) or editing a webhook. */
type Editing = { webhook: Webhook | null; n: number } | undefined;

function listOrAll(list: string[], all: string) {
  const l = list.filter((x) => x !== "*");
  return l.length ? l.join(", ") : all;
}

function LastDelivery({ w }: { w: Webhook }) {
  if (!w.lastDeliveryAt) return <Text color="gray.500">Never</Text>;
  const failed = deliveryFailed({
    statusCode: w.lastStatus,
    error: w.lastError,
  });
  return (
    <Box>
      <Badge colorPalette={failed ? "red" : "green"}>{w.lastStatus || "no response"}</Badge>
      <Text color="gray.600" fontSize="xs" whiteSpace="nowrap">
        {optTime(w.lastDeliveryAt)}
      </Text>
      {w.lastError && (
        <Text fontSize="xs" color="red.600" lineClamp={2} title={w.lastError} whiteSpace="normal">
          {w.lastError}
        </Text>
      )}
    </Box>
  );
}

export function WebhooksPage() {
  const notify = useNotify();
  const [editing, setEditing] = useState<Editing>(undefined);
  const [secret, setSecret] = useState<{ name: string; value: string } | null>(null);
  const [deliveriesOf, setDeliveriesOf] = useState<Webhook | null>(null);
  const [testing, setTesting] = useState<number | null>(null);
  const { data, loading, error, refetch } = useApiQuery("/admin/webhooks", async () => {
    const res = await getJSON<Webhook[]>("/admin/webhooks", "Failed to load webhooks");
    return Array.isArray(res.data) ? res.data : [];
  });
  const hooks = data ?? [];

  const openForm = (webhook: Webhook | null) => setEditing((e) => ({ webhook, n: (e?.n ?? 0) + 1 }));

  const remove = async (w: Webhook) => {
    try {
      await api("DELETE", `/admin/webhooks/${w.id}`);
      notify.success(`Webhook ${w.name} deleted`);
      refetch();
    } catch (err) {
      notify.failure(err, "Could not delete the webhook");
    }
  };

  const sendTest = async (w: Webhook) => {
    setTesting(w.id);
    try {
      const { data: r } = await api<WebhookTestResult>("POST", `/admin/webhooks/${w.id}/test`);
      const took = formatDuration(r.durationMs);
      if (r.delivered) {
        notify.success(`Test delivered to ${w.name}`, `HTTP ${r.statusCode} in ${took}`);
      } else {
        const status = r.statusCode ? `HTTP ${r.statusCode}` : "No response";
        notify.failure(new Error(`${status}${r.error ? `: ${r.error}` : ""} (${took})`), `Test delivery to ${w.name} failed`);
      }
    } catch (err) {
      notify.failure(err, "Could not send the test");
    } finally {
      setTesting(null);
    }
  };

  const newButton = (
    <Button colorPalette="blue" onClick={() => openForm(null)}>
      <FiPlus />
      New webhook
    </Button>
  );

  return (
    <Page>
      <PageHeader
        title="Webhooks"
        description="Send content events (item created, updated, published…) to another service: Slack, Zapier, a static-site rebuild hook. Failed deliveries are retried."
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
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} h="40px" />
          ))}
        </Stack>
      ) : data && hooks.length === 0 ? (
        <EmptyState icon={FiZap} title="No webhooks" action={newButton}>
          Add a webhook to notify another service whenever content changes.
        </EmptyState>
      ) : data ? (
        <Box bg="white" borderWidth="1px" borderRadius="lg" overflow="hidden">
          <Table.ScrollArea whiteSpace="normal">
            <Table.Root size="sm">
              <Table.Header>
                <Table.Row>
                  <Table.ColumnHeader>Webhook</Table.ColumnHeader>
                  <Table.ColumnHeader>Events</Table.ColumnHeader>
                  <Table.ColumnHeader>Collections</Table.ColumnHeader>
                  <Table.ColumnHeader>Last delivery</Table.ColumnHeader>
                  <Table.ColumnHeader textAlign="right">Actions</Table.ColumnHeader>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {hooks.map((w) => (
                  <Table.Row key={w.id}>
                    <Table.Cell maxW="300px">
                      <Text fontWeight="medium" lineClamp={1} title={w.name}>
                        {w.name}
                      </Text>
                      <Text fontSize="xs" color="gray.500" wordBreak="break-all" lineClamp={1} title={w.url}>
                        {w.url}
                      </Text>
                      <Wrap gap={1} mt={1}>
                        <Badge variant="outline">{w.format}</Badge>
                        {!w.enabled && <Badge colorPalette="orange">disabled</Badge>}
                        {w.signed && (
                          <Badge colorPalette="purple" title="Deliveries carry an X-Gohead-Signature">
                            signed
                          </Badge>
                        )}
                        {w.headers.length > 0 && (
                          <Badge title={`Custom headers: ${w.headers.join(", ")}`}>
                            {w.headers.length} header
                            {w.headers.length > 1 ? "s" : ""}
                          </Badge>
                        )}
                      </Wrap>
                    </Table.Cell>
                    <Table.Cell fontSize="xs">{listOrAll(w.events, "All events")}</Table.Cell>
                    <Table.Cell fontSize="xs">{listOrAll(w.collections, "All collections")}</Table.Cell>
                    <Table.Cell>
                      <LastDelivery w={w} />
                    </Table.Cell>
                    <Table.Cell>
                      <HStack gap={0} justify="flex-end">
                        <Tooltip content="Send test" showArrow>
                          <IconButton
                            aria-label={`Send test to ${w.name}`}
                            size="xs"
                            variant="ghost"
                            loading={testing === w.id}
                            onClick={() => sendTest(w)}
                          >
                            <FiSend />
                          </IconButton>
                        </Tooltip>
                        <Tooltip content="Deliveries" showArrow>
                          <IconButton aria-label={`Deliveries of ${w.name}`} size="xs" variant="ghost" onClick={() => setDeliveriesOf(w)}>
                            <FiList />
                          </IconButton>
                        </Tooltip>
                        <Tooltip content="Edit" showArrow>
                          <IconButton aria-label={`Edit ${w.name}`} size="xs" variant="ghost" onClick={() => openForm(w)}>
                            <FiEdit2 />
                          </IconButton>
                        </Tooltip>
                        <ConfirmButton
                          title={`Delete ${w.name}?`}
                          body="Events stop being sent to this URL and its delivery log is deleted. This cannot be undone."
                          onConfirm={() => remove(w)}
                          trigger={
                            <IconButton aria-label={`Delete ${w.name}`} size="xs" variant="ghost" colorPalette="red">
                              <FiTrash2 />
                            </IconButton>
                          }
                        />
                      </HStack>
                    </Table.Cell>
                  </Table.Row>
                ))}
              </Table.Body>
            </Table.Root>
          </Table.ScrollArea>
        </Box>
      ) : null}

      {editing && (
        <WebhookFormDrawer
          key={editing.n}
          isOpen
          webhook={editing.webhook}
          onClose={() => setEditing(undefined)}
          onSaved={(saved, created) => {
            notify.success(created ? `Webhook ${saved.name} created` : `Webhook ${saved.name} saved`);
            setEditing(undefined);
            if (saved.secret) setSecret({ name: saved.name, value: saved.secret });
            refetch();
          }}
        />
      )}

      <DeliveriesDrawer webhook={deliveriesOf} onClose={() => setDeliveriesOf(null)} />

      <SecretDialog
        title={secret ? `Signing secret for "${secret.name}"` : "Signing secret"}
        label="Signing secret"
        secret={secret?.value ?? null}
        onClose={() => setSecret(null)}
      >
        <Text fontSize="sm" color="gray.600">
          The receiver verifies <Code fontSize="xs">X-Gohead-Signature</Code>: <Code fontSize="xs">sha256=</Code> followed by the hex
          HMAC-SHA256 of <Code fontSize="xs">&lt;X-Gohead-Timestamp&gt;.&lt;body&gt;</Code> with this secret.
        </Text>
      </SecretDialog>
    </Page>
  );
}

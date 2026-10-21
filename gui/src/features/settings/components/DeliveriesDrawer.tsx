import { Alert, Badge, Button, HStack, Skeleton, Stack, Table, Text } from "@chakra-ui/react";
import { DrawerBody, DrawerCloseTrigger, DrawerContent, DrawerHeader, DrawerRoot, DrawerTitle } from "@/components/ui/drawer";
import { FiRefreshCw } from "react-icons/fi";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import { formatDuration, formatTime } from "../../agents/components/runFormat";
import { deliveryFailed, optTime } from "./settingsFormat";
import type { Webhook, WebhookDelivery } from "./types";

/** The latest delivery attempts of one webhook, failures highlighted. */
export function DeliveriesDrawer({ webhook, onClose }: { webhook: Webhook | null; onClose: () => void }) {
  const path = webhook ? `/admin/webhooks/${webhook.id}/deliveries?limit=100` : null;
  const { data, loading, error, refetch } = useApiQuery(path, async () => {
    const res = await getJSON<WebhookDelivery[]>(path ?? "", "Failed to load deliveries");
    return Array.isArray(res.data) ? res.data : [];
  });
  const list = data ?? [];
  const failures = list.filter(deliveryFailed).length;

  return (
    <DrawerRoot
      open={webhook !== null}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      size="xl"
    >
      <DrawerContent>
        <DrawerCloseTrigger />
        <DrawerHeader borderBottomWidth="1px" flexDirection="column" alignItems="stretch" gap={0}>
          <DrawerTitle>Deliveries: {webhook?.name}</DrawerTitle>
          <Text fontSize="sm" fontWeight="normal" color="gray.500" wordBreak="break-all">
            {webhook?.url}
          </Text>
        </DrawerHeader>
        <DrawerBody py={4}>
          <HStack justify="space-between" mb={3}>
            <Text fontSize="sm" color="gray.600">
              {data ? `Latest ${list.length} attempts${failures ? `, ${failures} failed` : ""}` : ""}
            </Text>
            <Button size="xs" variant="subtle" onClick={refetch} loading={loading && !!data}>
              <FiRefreshCw />
              Refresh
            </Button>
          </HStack>
          {error && (
            <Alert.Root status="error" borderRadius="md" mb={3}>
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Description>{error}</Alert.Description>
              </Alert.Content>
            </Alert.Root>
          )}
          {loading && !data ? (
            <Stack>
              {[0, 1, 2, 3].map((i) => (
                <Skeleton key={i} h="28px" />
              ))}
            </Stack>
          ) : data && list.length === 0 ? (
            <Text color="gray.500" py={8} textAlign="center">
              Nothing delivered yet. Deliveries appear here when a matching event happens; use “Send test” to try the URL now.
            </Text>
          ) : data ? (
            <Table.ScrollArea borderWidth="1px" borderRadius="lg" whiteSpace="normal">
              <Table.Root size="sm">
                <Table.Header>
                  <Table.Row>
                    <Table.ColumnHeader>Time</Table.ColumnHeader>
                    <Table.ColumnHeader>Event</Table.ColumnHeader>
                    <Table.ColumnHeader>Collection</Table.ColumnHeader>
                    <Table.ColumnHeader textAlign="end">Item</Table.ColumnHeader>
                    <Table.ColumnHeader textAlign="end">Attempt</Table.ColumnHeader>
                    <Table.ColumnHeader>Status</Table.ColumnHeader>
                    <Table.ColumnHeader textAlign="end">Duration</Table.ColumnHeader>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {list.map((d) => {
                    const failed = deliveryFailed(d);
                    return (
                      <Table.Row key={d.id} bg={failed ? "red.50" : undefined}>
                        <Table.Cell whiteSpace="nowrap" title={formatTime(d.createdAt)}>
                          {optTime(d.createdAt)}
                        </Table.Cell>
                        <Table.Cell whiteSpace="nowrap">{d.event}</Table.Cell>
                        <Table.Cell>{d.collection || "–"}</Table.Cell>
                        <Table.Cell textAlign="end">{d.itemId || "–"}</Table.Cell>
                        <Table.Cell textAlign="end">{d.attempt}</Table.Cell>
                        <Table.Cell>
                          <Badge colorPalette={failed ? "red" : "green"}>{d.statusCode || "no response"}</Badge>
                          {d.error && (
                            <Text fontSize="xs" color="red.600" mt={1} wordBreak="break-word">
                              {d.error}
                            </Text>
                          )}
                        </Table.Cell>
                        <Table.Cell textAlign="end" whiteSpace="nowrap">
                          {formatDuration(d.durationMs)}
                        </Table.Cell>
                      </Table.Row>
                    );
                  })}
                </Table.Body>
              </Table.Root>
            </Table.ScrollArea>
          ) : null}
        </DrawerBody>
      </DrawerContent>
    </DrawerRoot>
  );
}

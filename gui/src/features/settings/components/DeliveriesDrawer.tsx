import {
  Alert,
  AlertDescription,
  AlertIcon,
  Badge,
  Button,
  Drawer,
  DrawerBody,
  DrawerCloseButton,
  DrawerContent,
  DrawerHeader,
  DrawerOverlay,
  HStack,
  Skeleton,
  Stack,
  Table,
  TableContainer,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
} from "@chakra-ui/react";
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
    <Drawer isOpen={webhook !== null} onClose={onClose} size="xl">
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px">
          <Text>Deliveries: {webhook?.name}</Text>
          <Text fontSize="sm" fontWeight="normal" color="gray.500" wordBreak="break-all">
            {webhook?.url}
          </Text>
        </DrawerHeader>
        <DrawerBody py={4}>
          <HStack justify="space-between" mb={3}>
            <Text fontSize="sm" color="gray.600">
              {data ? `Latest ${list.length} attempts${failures ? `, ${failures} failed` : ""}` : ""}
            </Text>
            <Button size="sm" leftIcon={<FiRefreshCw />} onClick={refetch} isLoading={loading && !!data}>
              Refresh
            </Button>
          </HStack>
          {error && (
            <Alert status="error" borderRadius="md" mb={3}>
              <AlertIcon />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
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
            <TableContainer borderWidth="1px" borderRadius="lg" whiteSpace="normal">
              <Table size="sm">
                <Thead>
                  <Tr>
                    <Th>Time</Th>
                    <Th>Event</Th>
                    <Th>Collection</Th>
                    <Th isNumeric>Item</Th>
                    <Th isNumeric>Attempt</Th>
                    <Th>Status</Th>
                    <Th isNumeric>Duration</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {list.map((d) => {
                    const failed = deliveryFailed(d);
                    return (
                      <Tr key={d.id} bg={failed ? "red.50" : undefined}>
                        <Td whiteSpace="nowrap" title={formatTime(d.createdAt)}>
                          {optTime(d.createdAt)}
                        </Td>
                        <Td whiteSpace="nowrap">{d.event}</Td>
                        <Td>{d.collection || "–"}</Td>
                        <Td isNumeric>{d.itemId || "–"}</Td>
                        <Td isNumeric>{d.attempt}</Td>
                        <Td>
                          <Badge colorScheme={failed ? "red" : "green"}>{d.statusCode || "no response"}</Badge>
                          {d.error && (
                            <Text fontSize="xs" color="red.600" mt={1} wordBreak="break-word">
                              {d.error}
                            </Text>
                          )}
                        </Td>
                        <Td isNumeric whiteSpace="nowrap">
                          {formatDuration(d.durationMs)}
                        </Td>
                      </Tr>
                    );
                  })}
                </Tbody>
              </Table>
            </TableContainer>
          ) : null}
        </DrawerBody>
      </DrawerContent>
    </Drawer>
  );
}

import { Link as RouterLink } from "react-router-dom";
import { Badge, Box, Button, Flex, HStack, Text, VStack } from "@chakra-ui/react";
import type { Overview, RecentDelivery } from "../overview";
import { deliveryFailed, timeAgo } from "../overview";

function DeliveryRow({ d }: { d: RecentDelivery }) {
  const failed = deliveryFailed(d.statusCode);
  return (
    <Flex justify="space-between" align="center" gap={3} py={1.5} px={2} borderRadius="md" bg={failed ? "red.50" : undefined}>
      <Box minW={0}>
        <Text fontSize="sm" noOfLines={1}>
          {d.event}
          {d.collection && ` · ${d.collection}`}
          {d.itemId ? ` #${d.itemId}` : ""}
        </Text>
        {failed && d.error && (
          <Text fontSize="xs" color="red.600" noOfLines={1} title={d.error}>
            {d.error}
          </Text>
        )}
      </Box>
      <HStack spacing={2} flexShrink={0}>
        <Text fontSize="xs" color="gray.500">
          {timeAgo(d.createdAt)}
        </Text>
        <Badge colorScheme={failed ? "red" : "green"}>{d.statusCode || "no reply"}</Badge>
      </HStack>
    </Flex>
  );
}

/** Webhook deliveries of the last 24 hours, failures first in mind. */
export function DeliveriesCard({ webhooks, deliveries, recent }: { webhooks: number; deliveries: Overview["deliveries24h"]; recent: RecentDelivery[] }) {
  if (webhooks === 0) {
    return (
      <VStack align="start" spacing={3}>
        <Text fontSize="sm" color="gray.500">
          No webhooks yet. Webhooks notify other services when content changes.
        </Text>
        <Button as={RouterLink} to="/settings/webhooks" size="sm" variant="outline">
          Add a webhook
        </Button>
      </VStack>
    );
  }
  const failed = deliveries.failed;
  return (
    <>
      <HStack spacing={6} mb={3}>
        <Box>
          <Text fontSize="xs" color="gray.500">
            Last 24 h
          </Text>
          <Text fontSize="xl" fontWeight="bold">
            {deliveries.total.toLocaleString()}
          </Text>
        </Box>
        <Box>
          <Text fontSize="xs" color="gray.500">
            Failed
          </Text>
          <Text fontSize="xl" fontWeight="bold" color={failed > 0 ? "red.500" : undefined}>
            {failed.toLocaleString()}
          </Text>
        </Box>
      </HStack>
      {recent.length === 0 ? (
        <Text fontSize="sm" color="gray.500">
          Nothing delivered yet.
        </Text>
      ) : (
        <VStack align="stretch" spacing={0.5}>
          {recent.slice(0, 5).map((d) => (
            <DeliveryRow key={d.id} d={d} />
          ))}
        </VStack>
      )}
    </>
  );
}

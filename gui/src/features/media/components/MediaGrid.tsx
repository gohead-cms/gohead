import { Box, SimpleGrid, Text } from "@chakra-ui/react";
import { type MediaItem, dimensions, formatBytes } from "../media";
import { MediaThumb } from "./MediaThumb";

function MediaCard({ item, onOpen }: { item: MediaItem; onOpen: () => void }) {
  const dims = dimensions(item);
  return (
    <Box
      as="button"
      type="button"
      onClick={onOpen}
      aria-label={`Open ${item.name}`}
      textAlign="left"
      bg="white"
      borderWidth="1px"
      borderRadius="lg"
      overflow="hidden"
      minW={0}
      transition="box-shadow 0.15s, border-color 0.15s"
      _hover={{ shadow: "md", borderColor: "blue.300" }}
      _focusVisible={{ outline: "2px solid", outlineColor: "blue.400", outlineOffset: "2px" }}
    >
      <MediaThumb item={item} height="130px" />
      <Box px={3} py={2}>
        <Text fontSize="sm" fontWeight="medium" noOfLines={1} title={item.name}>
          {item.name}
        </Text>
        <Text fontSize="xs" color="gray.500" noOfLines={1}>
          {formatBytes(item.size)}
          {dims && ` · ${dims}`}
        </Text>
      </Box>
    </Box>
  );
}

/** Responsive grid of media cards. */
export function MediaGrid({ items, onOpen }: { items: MediaItem[]; onOpen: (id: number) => void }) {
  return (
    <SimpleGrid minChildWidth="160px" spacing={4}>
      {items.map((m) => (
        <MediaCard key={m.id} item={m} onOpen={() => onOpen(m.id)} />
      ))}
    </SimpleGrid>
  );
}

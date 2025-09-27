import React from "react";
import { Box, Text, VStack, ListItem, List, Spinner, useColorModeValue } from "@chakra-ui/react";
import { Schema } from "../../../shared/types";

interface CollectionSelectorProps {
  collections: { schema: Schema }[];
  selectedCollection: string | null;
  onSelect: (name: string) => void;
  isLoading: boolean;
}

export function CollectionSelector({
  collections,
  selectedCollection,
  onSelect,
  isLoading,
}: CollectionSelectorProps) {
  // Call all hooks at the top level, before any conditional returns.
  const sidebarBg = useColorModeValue("white", "gray.800");
  const activeBg = useColorModeValue("purple.50", "purple.900");
  const activeColor = useColorModeValue("purple.700", "white");
  const hoverBg = useColorModeValue("gray.100", "gray.700");

  // It is now safe to return early for the loading state.
  if (isLoading) {
    return (
        <Box
            w="280px"
            bg={sidebarBg}
            borderRight="1px solid"
            borderColor={useColorModeValue("gray.200", "gray.700")}
            p={4}
        >
            <Text fontSize="lg" fontWeight="bold" mb={4}>
                Collections
            </Text>
            <Spinner />
        </Box>
    );
  }

  return (
    <Box
      w="280px"
      bg={sidebarBg}
      borderRight="1px solid"
      borderColor={useColorModeValue("gray.200", "gray.700")}
      p={4}
    >
      <Text fontSize="lg" fontWeight="bold" mb={4}>
        Collections
      </Text>
      <List spacing={1}>
        {collections.map(({ schema }) => (
          <ListItem
            key={schema.collectionName}
            onClick={() => onSelect(schema.collectionName!)}
            py={2}
            px={3}
            borderRadius="md"
            cursor="pointer"
            bg={selectedCollection === schema.collectionName ? activeBg : "transparent"}
            color={selectedCollection === schema.collectionName ? activeColor : "inherit"}
            fontWeight={selectedCollection === schema.collectionName ? "semibold" : "normal"}
            _hover={{ bg: hoverBg }}
          >
            {schema.info?.displayName || schema.collectionName}
          </ListItem>
        ))}
      </List>
    </Box>
  );
}


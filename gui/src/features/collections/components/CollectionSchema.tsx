import React from "react";
import {
  Box,
  Heading,
  Text,
  Flex,
  Button,
  Spinner,
  HStack,
  Alert,
  AlertIcon
} from "@chakra-ui/react";
import { FiEdit2, FiPlus, FiSave } from "react-icons/fi";
import { useCollectionSchema } from "../../collections/hook/useCollectionSchema";
import { AttributesTable } from "./AttributesTable";

export default function CollectionSchema({ collectionName }: { collectionName: string }) {
  // The component now uses the clean hook for its data and state
  const { schema, loading, error } = useCollectionSchema(collectionName);

  if (loading) return <Spinner />;
  if (error) return <Alert status="error"><AlertIcon />{error}</Alert>;
  if (!schema) return <Text color="gray.500">Select a collection to view its schema.</Text>;

  return (
    <Box>
      {/* Header section */}
      <Flex align="center" mb={4} gap={4} justify="space-between">
        <HStack>
          <Heading size="xl">
            {schema.info?.displayName || schema.collectionName}
          </Heading>
          <Button size="sm" variant="outline" leftIcon={<FiEdit2 />}>
            Edit
          </Button>
        </HStack>
        <HStack>
          <Button colorScheme="blue" variant="outline" size="sm" leftIcon={<FiPlus />}>
            Add field
          </Button>
          <Button colorScheme="gray" variant="outline" size="sm" leftIcon={<FiSave />}>
            Save
          </Button>
        </HStack>
      </Flex>

      <Text color="gray.500" mb={8}>
        Build the data architecture of your content.
      </Text>

      {/* The complex table logic is now neatly encapsulated */}
      <AttributesTable attributes={schema.attributes} />

      <Button mt={6} colorScheme="blue" variant="ghost" size="sm" leftIcon={<FiPlus />}>
        Add another field to this collection
      </Button>
    </Box>
  );
}
import React, { useEffect, useState } from "react";
import { Flex, Box, Spinner, Text, Badge, Link, Icon, List, ListItem } from "@chakra-ui/react";
import {
  FiPlus} from "react-icons/fi";
import { apiFetchWithAuth } from "../../../services/api";
import CollectionSchema from "./CollectionSchema";

export function CollectionsPage() {
  const [collections, setCollections] = useState<{ name: string }[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [schema, setSchema] = useState<any | null>(null);
  const [loading, setLoading] = useState(true);
  const [schemaLoading, setSchemaLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    apiFetchWithAuth("/admin/collections")
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch collections");
        const json = await res.json();
        const list = Array.isArray(json.data) ? json.data : [];
        setCollections(list.map((c: any) => ({ name: c.schema.collectionName })));
        setSelected(list.length ? list[0].schema.collectionName : null);
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    if (!selected) return;
    setSchemaLoading(true);
    apiFetchWithAuth(`/admin/collections/${selected}`)
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch collection schema");
        const json = await res.json();
        setSchema(json.data?.schema || null);
      })
      .finally(() => setSchemaLoading(false));
  }, [selected]);

  return (
    <Flex h="100%" w="100%">
      <Box
        w="320px"
        borderRight="1px solid"
        borderColor="gray.200"
        overflowY="auto"
        px={6}                // Add horizontal padding
        py={8}                // Add vertical padding
        bg="gray.100"          // Light background for sidebar
      >
       <Text
          mb={2}
          fontWeight="bold"
          fontSize="sm"
          letterSpacing="wide"
          textTransform="uppercase"
        >
          Collections
        <Badge 
          variant="surface"
          colorScheme="cyan" // Corrected prop from colorPalette
          fontSize="0.75em"
          borderRadius="full"
          px={2}
          py={0.5}
          m={2}
          fontWeight="normal"
        >
          {collections.length}
        </Badge>

        </Text>
        {loading ? (
          <Spinner />
        ) : (
          <List> {/* Corrected List.Root to List */}
            {collections.map(({ name }) => (
              <ListItem // Corrected List.Item to ListItem
                key={name}
                cursor="pointer"
                py={1}
                px={5}
                ms={2}
                fontWeight={selected === name ? "bold" : "normal"}
                borderRadius="lg"
                onClick={() => setSelected(name)}
                _before={{
                  display: "inline-block",
                  color: "gray.400",
                  fontSize: "2xl",
                  marginRight: "30px",
                  marginLeft: "20px"
                }}
              >
                 {name.charAt(0).toUpperCase() + name.slice(1)}
              </ListItem>
            ))}
          </List>
        )}
         <Box mt={6} px={3}>
        <Link
          color="blue.500"
          fontWeight="semibold"
          fontSize="md"
          display="flex"
          alignItems="center"
          gap={2}
          cursor="pointer"
          href="/collections/studio" 
          _hover={{ textDecoration: "underline" }}
        >
          <Icon as={FiPlus} boxSize={4} mr={1} />
          Create Collection
        </Link>
      </Box>
      </Box>
      
      <Box
        flex="1"
        p={10}
        overflowY="auto"
        bg="gray.50"
        minH="100%"
      >
        {schemaLoading ? (
          <Spinner />
        ) : selected && schema ? (
          <CollectionSchema collectionName={schema.collectionName} />
        ) : (
          <Text color="gray.400">Select a collection to view its schema</Text>
        )}
      </Box>
    </Flex>
  );
}

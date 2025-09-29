import React, { useEffect, useState } from "react";
import { Flex, Box, Heading, Text, Button, Icon, useDisclosure, useToast, HStack } from "@chakra-ui/react";
import { FiPlus } from "react-icons/fi";
import { 
  useCollectionsList,
  useCollectionContent
} from "../hooks";
import { CollectionSelector, ContentTable, EntryFormModal } from ".";
import { PageLoader } from "../../../shared/ui/page-loader";
import { apiFetchWithAuth } from "../../../services/api";
import { ContentItem } from "../hooks/useCollectionContent";
import { useNavigate } from "react-router-dom";
import { FaPenNib } from "react-icons/fa6";
import { CiPen } from "react-icons/ci";

export function ContributionsPage() {
  const [selectedCollection, setSelectedCollection] = useState<string | null>(null);
  const { isOpen, onOpen, onClose } = useDisclosure(); // For the form drawer
  const toast = useToast();
  const navigate = useNavigate();

  const { collections, loading: collectionsLoading } = useCollectionsList();
  const { content, loading: contentLoading, refetch } = useCollectionContent(selectedCollection);

  const [editingItem, setEditingItem] = useState<ContentItem | null>(null);

  useEffect(() => {
    if (!selectedCollection && collections.length > 0) {
      const firstCollectionName = collections[0].schema.collectionName;
      if (firstCollectionName) {
        setSelectedCollection(firstCollectionName);
      }
    }
  }, [collections, selectedCollection]);

  const currentCollectionSchema = collections.find(c => c.schema.collectionName === selectedCollection)?.schema;

  const handleAddItem = async (data: Partial<ContentItem>) => {
    if (!selectedCollection) return;

    try {
      const response = await apiFetchWithAuth(`/api/collections/${selectedCollection}`, {
        method: 'POST',
        body: JSON.stringify(data),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.error?.message || "Failed to create entry.");
      }
      
      toast({
        title: "Entry Created",
        description: "The new entry was saved successfully.",
        status: "success",
        duration: 5000,
        isClosable: true,
      });

      await refetch();
    } catch (error: any) {
      toast({
        title: "An error occurred.",
        description: error.message,
        status: "error",
        duration: 5000,
        isClosable: true,
      });
      throw error;
    }
  };

  const handleContribClick = () => {
    if (selectedCollection) {
      // Navigate to the dedicated creative editor page
      navigate(`/contrib/edit/${selectedCollection}`);
    }
  };

  const handleEditClick = (item: ContentItem) => {
    setEditingItem(item);
    onOpen(); 
  };

  return (
    <Flex h="calc(100vh - 64px)">
      <CollectionSelector
        collections={collections}
        selectedCollection={selectedCollection}
        onSelect={setSelectedCollection}
        isLoading={collectionsLoading}
      />

      <Box flex="1" p={8} overflowY="auto">
        {(contentLoading && !isOpen) && <PageLoader text="Fetching content..." />}

        {!contentLoading && currentCollectionSchema && (
          <>
            <Flex justify="space-between" align="center" mb={6}>
              <Box>
                <Heading size="lg">
                  {currentCollectionSchema.info?.displayName || currentCollectionSchema.collectionName}
                </Heading>
                <Text color="gray.500">{content.items.length} entries found</Text>
              </Box>
              {/* Button Group for adding content */}
              <HStack>
                <Button bg="white" variant="outline"  size="sm" leftIcon={<Icon as={FiPlus} />} onClick={onOpen}>
                  Quick Add
                </Button>
                <Button colorScheme="purple" size="sm" leftIcon={<Icon as={FaPenNib} />} onClick={handleContribClick}>
                  Contrib
                </Button>
              </HStack>
            </Flex>
            <ContentTable
              schema={currentCollectionSchema}
              items={content.items}
              onEdit={handleEditClick}
            />
          </>
        )}

        {currentCollectionSchema && (
            <EntryFormModal
                isOpen={isOpen}
                onClose={onClose}
                schema={currentCollectionSchema}
                onSubmit={handleAddItem}
            />
        )}

        {!selectedCollection && !collectionsLoading && (
            <Flex justify="center" align="center" h="100%">
                <Text color="gray.400">Select a collection to browse its content.</Text>
            </Flex>
        )}
      </Box>
    </Flex>
  );
}


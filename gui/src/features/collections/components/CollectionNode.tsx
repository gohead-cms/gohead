import React from 'react';
import { Handle, NodeProps, Position } from '@xyflow/react';
import { Box, VStack, HStack, Text, Icon, Divider } from '@chakra-ui/react';
import { FaDatabase, FaQuoteLeft, FaHashtag, FaToggleOn, FaLink, FaCalendarAlt, FaCheckSquare, FaImage } from 'react-icons/fa';
import { CollectionNodeData, AttributeItem, CollectionNodeType } from '../../../shared/types';

// Map attribute types to specific icons for a better visual representation
const attributeIcons: Record<string, React.ElementType> = {
  string: FaQuoteLeft,
  text: FaQuoteLeft,
  number: FaHashtag,
  integer: FaHashtag,
  float: FaHashtag,
  boolean: FaToggleOn,
  relation: FaLink,
  date: FaCalendarAlt,
  datetime: FaCalendarAlt,
  richtext: FaQuoteLeft,
  media: FaImage, // Added new icon for media type
  // Add other types as needed
};

const getAttributeIcon = (type: string) => {
  // Handle relation type specifically to extract the base type
  const baseType = type.toLowerCase().includes('relation') ? 'relation' : type.toLowerCase();
  return attributeIcons[baseType] || FaCheckSquare; // Default icon
};

export function CollectionNode({ data, selected }: NodeProps<CollectionNodeType>) {
  const attributes = Array.isArray(data?.attributes) ? data.attributes : [];

  return (
    <>
      <Handle type="target" position={Position.Left} style={{ background: '#52b4ca' }} />
      <Box
        bg="white"
        borderRadius="lg"
        border="2px solid"
        borderColor={selected ? '#52b4ca' : '#e2e8f0'}
        boxShadow={selected ? '0 0 0 3px rgba(82, 180, 202, 0.2)' : 'sm'}
        minW={240}
        maxW={280}
        overflow="hidden"
        transition="all 0.2s"
      >
        {/* Header */}
        <HStack
          bg="#52b4ca" // Reverted to the original solid blue color
          color="white"
          p={3}
          spacing={3}
        >
          <Icon as={FaDatabase} boxSize={5} />
          <Text fontWeight="bold" fontSize="md" noOfLines={1}>
            {data.label}
          </Text>
        </HStack>

        {/* Body with Attributes List */}
        <VStack spacing={2} align="stretch" p={3} maxH="200px" overflowY="auto">
          {attributes.length > 0 ? (
            attributes.map((attr: AttributeItem) => (
              <React.Fragment key={attr.name}>
                <HStack
                  justify="space-between"
                  spacing={3}
                  _hover={{ bg: 'gray.50' }}
                  p={1}
                  borderRadius="md"
                >
                  <HStack>
                    <Icon
                      as={getAttributeIcon(attr.type)}
                      color="gray.400"
                      boxSize={3}
                    />
                    <Text fontSize="sm" fontWeight="medium" color="gray.700">
                      {attr.name}
                    </Text>
                  </HStack>
                  <Text fontSize="xs" color="gray.500" noOfLines={1} maxW="100px">
                    {attr.type}
                  </Text>
                </HStack>
                <Divider />
              </React.Fragment>
            ))
          ) : (
            <Text fontSize="sm" color="gray.400" fontStyle="italic" textAlign="center" p={2}>
              No attributes defined
            </Text>
          )}
        </VStack>
      </Box>
      <Handle type="source" position={Position.Right} style={{ background: '#52b4ca' }} />
    </>
  );
}


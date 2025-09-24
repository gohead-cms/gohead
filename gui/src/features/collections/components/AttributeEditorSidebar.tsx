import React, { useState, useEffect } from 'react';
import {
  Drawer,
  DrawerBody,
  DrawerFooter,
  DrawerHeader,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  Button,
  VStack,
  FormControl,
  FormLabel,
  Input,
  Select,
  IconButton,
  HStack,
  Box,
  Text,
  useToast,
} from '@chakra-ui/react';
import { FaTrash, FaPlus } from 'react-icons/fa';
import { Node } from "@xyflow/react";

// Define a local type for the attributes to avoid conflicts
type AttributeItem = {
  name: string;
  type: string;
};

// Props for the sidebar component
type AttributeEditorProps = {
  isOpen: boolean;
  onClose: () => void;
  node: Node<{ label: string; attributes: AttributeItem[] }> | null;
  onSave: (nodeId: string, newAttributes: AttributeItem[]) => void;
};

export const AttributeEditorSidebar: React.FC<AttributeEditorProps> = ({
  isOpen,
  onClose,
  node,
  onSave,
}) => {
  const toast = useToast();
  const [currentAttributes, setCurrentAttributes] = useState<AttributeItem[]>([]);
  const [newAttribute, setNewAttribute] = useState({ name: '', type: 'Text' });

  // Update attributes state when the node changes
  useEffect(() => {
    if (node) {
      setCurrentAttributes(node.data.attributes);
    }
  }, [node]);

  const handleAddAttribute = () => {
    if (!newAttribute.name.trim()) {
      toast({
        title: "Name is required.",
        status: "error",
        duration: 3000,
        isClosable: true,
      });
      return;
    }
    const updatedAttributes = [...currentAttributes, newAttribute];
    setCurrentAttributes(updatedAttributes);
    setNewAttribute({ name: '', type: 'Text' });
    onSave(node!.id, updatedAttributes);
  };

  const handleDeleteAttribute = (index: number) => {
    const updatedAttributes = currentAttributes.filter((_, i) => i !== index);
    setCurrentAttributes(updatedAttributes);
    onSave(node!.id, updatedAttributes);
  };
  
  const handleFinalSave = () => {
    toast({
        title: "Collection saved.",
        description: `Attributes for "${node?.data.label}" have been updated.`,
        status: "success",
        duration: 5000,
        isClosable: true,
      });
      onClose();
  }

  if (!node) {
    return null;
  }

  return (
    <Drawer isOpen={isOpen} placement="right" onClose={onClose} size="sm">
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px">
          Edit Collection: {node.data.label}
        </DrawerHeader>

        <DrawerBody>
          <VStack spacing={4} align="stretch">
            <Text fontWeight="bold" fontSize="lg">
              Attributes
            </Text>
            {currentAttributes.length > 0 ? (
              currentAttributes.map((attr, index) => (
                <HStack key={index} w="100%" justifyContent="space-between" p={2} borderWidth="1px" borderRadius="md">
                  <Box>
                    <Text fontWeight="medium">{attr.name}</Text>
                    <Text fontSize="sm" color="gray.500">{attr.type}</Text>
                  </Box>
                  <IconButton
                    size="sm"
                    aria-label="Delete attribute"
                    icon={<FaTrash />}
                    colorScheme="red"
                    onClick={() => handleDeleteAttribute(index)}
                  />
                </HStack>
              ))
            ) : (
              <Text color="gray.500">No attributes yet. Add one below!</Text>
            )}

            <Box borderTopWidth="1px" pt={4}>
              <Text fontWeight="bold" fontSize="lg" mb={2}>
                Add new attribute
              </Text>
              <HStack spacing={2}>
                <FormControl flex="1">
                  <FormLabel>Name</FormLabel>
                  <Input
                    placeholder="e.g., Title"
                    value={newAttribute.name}
                    onChange={(e) => setNewAttribute({ ...newAttribute, name: e.target.value })}
                  />
                </FormControl>
                <FormControl flex="1">
                  <FormLabel>Type</FormLabel>
                  <Select
                    value={newAttribute.type}
                    onChange={(e) => setNewAttribute({ ...newAttribute, type: e.target.value })}
                  >
                    <option value="Text">Text</option>
                    <option value="Number">Number</option>
                    <option value="Boolean">Boolean</option>
                    <option value="Relation">Relation</option>
                  </Select>
                </FormControl>
                <IconButton
                  mt={7}
                  size="md"
                  aria-label="Add attribute"
                  icon={<FaPlus />}
                  colorScheme="teal"
                  onClick={handleAddAttribute}
                  isDisabled={!newAttribute.name}
                />
              </HStack>
            </Box>
          </VStack>
        </DrawerBody>

        <DrawerFooter borderTopWidth="1px">
          <Button variant="outline" mr={3} onClick={onClose}>
            Cancel
          </Button>
          <Button colorScheme="blue" onClick={handleFinalSave}>
            Finish
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
};
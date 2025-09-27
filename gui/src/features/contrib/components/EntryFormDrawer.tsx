import React from 'react';
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalCloseButton,
  ModalBody,
  ModalFooter,
  Button,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Switch,
  VStack,
  Grid,
  GridItem,
  Heading,
  Text,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
  Box,
  Flex,
  HStack,
} from "@chakra-ui/react";
import { Schema } from "../../../shared/types";
import { ContentItem } from "../hooks/useCollectionContent";
import { useState, useEffect } from "react";

interface EntryFormDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  schema: Schema;
  onSubmit: (data: Partial<ContentItem>) => Promise<void>;
}

// Helper to render form fields in the sidebar
const renderSidebarField = (key: string, attr: any, value: any, handleChange: any) => {
  const commonProps = {
    id: key,
    value: value || '',
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => handleChange(key, e.target.value),
    size: 'sm',
  };

  switch (attr.type) {
    case 'number':
      return <Input type="number" {...commonProps} />;
    case 'date':
      return <Input type="date" {...commonProps} />;
    case 'boolean':
      return (
        <Flex justify="space-between" align="center">
          <FormLabel htmlFor={key} mb="0" fontSize="sm">
            Enable
          </FormLabel>
          <Switch
            id={key}
            isChecked={value || false}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleChange(key, e.target.checked)}
          />
        </Flex>
      );
    default: // Catches 'string', 'text', etc. for the sidebar
      return <Input type="text" {...commonProps} />;
  }
};

export function EntryFormDrawer({ isOpen, onClose, schema, onSubmit }: EntryFormDrawerProps) {
  const [formData, setFormData] = useState<Partial<ContentItem>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Separate fields for the editor layout
  const [mainContentField, setMainContentField] = useState<{ key: string, attr: any } | null>(null);
  const [titleField, setTitleField] = useState<{ key: string, attr: any } | null>(null);
  const [sidebarFields, setSidebarFields] = useState<Record<string, any>>({});

  useEffect(() => {
    if (isOpen) {
      const attributes = schema.attributes;
      let mainField: { key: string, attr: any } | null = null;
      let title: { key: string, attr: any } | null = null;
      const sidebar: Record<string, any> = {};
      const initialData: Partial<ContentItem> = {};

      // Heuristic to sort fields into layout positions
      Object.entries(attributes).forEach(([key, attr]) => {
        if (attr.type === 'relation') return;

        if (!title && key.toLowerCase() === 'title' && attr.type !== 'richtext' && attr.type !== 'text') {
          title = { key, attr };
        } else if (!mainField && (attr.type === 'richtext' || attr.type === 'text')) {
          mainField = { key, attr };
        } else {
          sidebar[key] = attr;
        }

        // Set initial data
        initialData[key] = attr.type === 'boolean' ? false : '';
      });
      
      setTitleField(title);
      setMainContentField(mainField);
      setSidebarFields(sidebar);
      setFormData(initialData);
    }
  }, [isOpen, schema]);

  const handleChange = (key: string, value: any) => {
    setFormData(prev => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
    try {
      await onSubmit(formData);
      onClose();
    } catch (error) {
      // Parent component handles error toast
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="full">
      <ModalOverlay />
      <ModalContent bg="gray.50">
        <ModalHeader borderBottomWidth="1px" bg="white">
          <Flex justify="space-between" align="center">
            <Box>
              <Heading size="md">{schema.info?.displayName || schema.collectionName}</Heading>
              <Text fontSize="sm" color="gray.500" fontWeight="normal">Editing new entry</Text>
            </Box>
            <HStack>
              <Button variant="ghost" mr={3} onClick={onClose} isDisabled={isSubmitting}>Cancel</Button>
              <Button colorScheme="purple" onClick={handleSubmit} isLoading={isSubmitting}>Save Entry</Button>
            </HStack>
          </Flex>
          <ModalCloseButton />
        </ModalHeader>
        <ModalBody>
          <Grid templateColumns={{ base: '1fr', lg: '3fr 1fr' }} gap={8} maxW="1200px" mx="auto" pt={8}>
            {/* Main Content Area */}
            <GridItem as={VStack} spacing={6} align="stretch">
              {titleField && (
                <FormControl isRequired={titleField.attr.required}>
                  <Input
                    id={titleField.key}
                    value={formData[titleField.key] as string || ''}
                    onChange={(e) => handleChange(titleField.key, e.target.value)}
                    placeholder="Your post title..."
                    variant="unstyled"
                    fontSize="4xl"
                    fontWeight="bold"
                    h="auto"
                    py={2}
                  />
                </FormControl>
              )}
              {mainContentField && (
                <FormControl isRequired={mainContentField.attr.required}>
                  <Textarea
                    id={mainContentField.key}
                    value={formData[mainContentField.key] as string || ''}
                    onChange={(e) => handleChange(mainContentField.key, e.target.value)}
                    placeholder="Begin writing your content here..."
                    variant="unstyled"
                    fontSize="lg"
                    rows={20}
                    p={2}
                  />
                </FormControl>
              )}
            </GridItem>
            
            {/* Sidebar for Metadata */}
            <GridItem as={VStack} spacing={4} align="stretch">
              <Accordion allowToggle defaultIndex={[0]}>
                <AccordionItem borderWidth="0" bg="white" borderRadius="lg" boxShadow="sm">
                  <AccordionButton>
                    <Box flex="1" textAlign="left" fontWeight="semibold">Metadata</Box>
                    <AccordionIcon />
                  </AccordionButton>
                  <AccordionPanel pb={4}>
                    <VStack spacing={4} align="stretch">
                      {Object.entries(sidebarFields).map(([key, attr]) => (
                        <FormControl key={key} isRequired={attr.required}>
                          <FormLabel htmlFor={key} fontSize="sm" fontWeight="medium">{key}</FormLabel>
                          {renderSidebarField(key, attr, formData[key], handleChange)}
                        </FormControl>
                      ))}
                    </VStack>
                  </AccordionPanel>
                </AccordionItem>
              </Accordion>
            </GridItem>
          </Grid>
        </ModalBody>
      </ModalContent>
    </Modal>
  );
}


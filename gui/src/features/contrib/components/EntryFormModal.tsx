import React, { useState, useEffect } from "react";
import {
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerHeader,
  DrawerCloseButton,
  DrawerBody,
  DrawerFooter,
  Button,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Switch,
  VStack,
} from "@chakra-ui/react";
import { Schema } from "../../../shared/types";
import { ContentItem } from "../hooks/useCollectionContent";

interface EntryFormDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  schema: Schema;
  onSubmit: (data: Partial<ContentItem>) => Promise<void>;
}

// Helper function to render the correct form field based on attribute type
const renderFormField = (key: string, attr: any, value: any, handleChange: any) => {
  const commonProps = {
    id: key,
    value: value || '',
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => handleChange(key, e.target.value),
  };

  switch (attr.type) {
    case 'text':
      return <Textarea {...commonProps} placeholder={`Enter ${key}`} />;
    case 'richtext':
      return <Textarea {...commonProps} placeholder={`Enter ${key}`} rows={8} />;
    case 'number':
      return <Input type="number" {...commonProps} />;
    case 'date':
        return <Input type="date" {...commonProps} />;
    case 'boolean':
      return (
        <Switch
          id={key}
          isChecked={value || false}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleChange(key, e.target.checked)}
        />
      );
    case 'string':
    default:
      return <Input type="text" {...commonProps} placeholder={`Enter ${key}`} />;
  }
};

export function EntryFormModal({ isOpen, onClose, schema, onSubmit }: EntryFormDrawerProps) {
  const [formData, setFormData] = useState<Partial<ContentItem>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (isOpen) {
      const initialData = Object.entries(schema.attributes).reduce((acc, [key, attr]) => {
          if (attr.type === 'relation') return acc;
          acc[key] = attr.type === 'boolean' ? false : '';
          return acc;
      }, {} as any);
      setFormData(initialData);
    }
  }, [isOpen, schema]);

  const handleChange = (key: string, value: string | boolean) => {
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
    <Drawer isOpen={isOpen} placement="right" onClose={onClose} size="lg">
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px">
          Add New Entry to {schema.info?.displayName || schema.collectionName}
        </DrawerHeader>
        <DrawerBody>
          <VStack spacing={6}>
            {Object.entries(schema.attributes).map(([key, attr]) => {
              if (attr.type === 'relation') return null;
              
              return (
                <FormControl key={key} isRequired={attr.required}>
                  <FormLabel htmlFor={key}>{key}</FormLabel>
                  {renderFormField(key, attr, formData[key], handleChange)}
                </FormControl>
              );
            })}
          </VStack>
        </DrawerBody>
        <DrawerFooter borderTopWidth="1px">
          <Button variant="outline" mr={3} onClick={onClose} isDisabled={isSubmitting}>
            Cancel
          </Button>
          <Button colorScheme="purple" onClick={handleSubmit} isLoading={isSubmitting}>
            Save Entry
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}

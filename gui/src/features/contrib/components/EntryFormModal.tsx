import React, { useState } from "react";
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
  Select,
  VStack,
} from "@chakra-ui/react";
import { Attribute, Schema } from "../../../shared/types";
import { ContentItem } from "../hooks/useCollectionContent";
import { MediaField, MediaValue } from "./MediaField";
import { toMediaRef } from "./mediaRef";
import { blankValue, inputType, isBooleanType, toItemData } from "./entryValues";

interface EntryFormDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  schema: Schema;
  onSubmit: (payload: { data: Partial<ContentItem> }, id?: string | number) => Promise<void>;
  itemToEdit?: ContentItem | null; // Optional prop for the item being edited
}

// Helper function to render the correct form field based on attribute type
const renderFormField = (key: string, attr: Attribute, value: unknown, handleChange: (key: string, value: unknown) => void) => {
  const commonProps = {
    id: key,
    value: typeof value === 'string' || typeof value === 'number' ? value : '',
    onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => handleChange(key, e.target.value),
  };

  if (isBooleanType(attr.type)) {
    return (
      <Switch
        id={key}
        isChecked={value === true}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleChange(key, e.target.checked)}
      />
    );
  }
  if (attr.type === 'enum' && attr.options?.length) {
    return (
      <Select id={key} value={typeof value === 'string' ? value : ''} placeholder="Choose…"
        onChange={(e) => handleChange(key, e.target.value)}>
        {attr.options.map((o) => <option key={o} value={o}>{o}</option>)}
      </Select>
    );
  }
  switch (attr.type) {
    case 'media':
      return (
        <MediaField
          id={key}
          value={value as MediaValue | MediaValue[]}
          multiple={attr.multiple}
          accept={attr.allowed_types}
          onChange={(v) => handleChange(key, v)}
        />
      );
    case 'text':
      return <Textarea {...commonProps} placeholder={`Enter ${key}`} />;
    case 'richtext':
      return <Textarea {...commonProps} placeholder={`Enter ${key}`} rows={8} />;
    default:
      return <Input type={inputType(attr.type)} {...commonProps} placeholder={`Enter ${key}`} />;
  }
};

// The form's starting values: the item's attributes when editing (items are
// flat; system fields are dropped), blank values when creating.
function initialFormData(schema: Schema, itemToEdit?: ContentItem | null): Partial<ContentItem> {
  if (itemToEdit) {
    const systemFields = ['id', 'createdAt', 'updatedAt', 'publishedAt', 'status'];
    return Object.fromEntries(
      Object.entries(itemToEdit).filter(([key]) => !systemFields.includes(key) || key in schema.attributes),
    );
  }
  return Object.entries(schema.attributes).reduce((acc, [key, attr]) => {
    if (attr.type === 'relation') return acc;
    if (attr.type === 'media') {
      if (attr.multiple) acc[key] = [];
      return acc;
    }
    acc[key] = blankValue(attr);
    return acc;
  }, {} as Record<string, unknown>);
}

/**
 * Creates or edits an entry. The form starts from schema and itemToEdit when
 * it mounts: give it a new key each time it is opened for another entry.
 */
export function EntryFormModal({ isOpen, onClose, schema, itemToEdit, onSubmit }: EntryFormDrawerProps) {
  const [formData, setFormData] = useState<Partial<ContentItem>>(() => initialFormData(schema, itemToEdit));
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleChange = (key: string, value: unknown) => {
    setFormData(prev => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
try {
      // Empty fields are left out and numbers converted; media fields hold
      // uploaded files, and the API wants their IDs.
      const data = toItemData(schema.attributes, formData);
      Object.entries(schema.attributes).forEach(([key, attr]) => {
        if (attr.type !== 'media') return;
        if (data[key] === null || data[key] === undefined) delete data[key];
        else data[key] = toMediaRef(data[key]);
      });
      await onSubmit({ data: data as Partial<ContentItem> }, itemToEdit?.id);
      onClose();
    } catch {
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
          {itemToEdit ? "Edit Entry" : "Add New Entry"} in {schema.info?.displayName || schema.collectionName}
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
            Save Changes
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}

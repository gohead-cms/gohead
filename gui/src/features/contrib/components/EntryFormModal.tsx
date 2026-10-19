import { useState } from "react";
import {
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerHeader,
  DrawerCloseButton,
  DrawerBody,
  DrawerFooter,
  Button,
} from "@chakra-ui/react";
import type { Schema } from "../../../shared/types";
import type { ContentItem } from "../hooks/useCollectionContent";
import { EntryFields } from "./EntryFields";
import { initialValues, toEntryPayload } from "./entryValues";

interface EntryFormDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  schema: Schema;
  onSubmit: (payload: { data: Partial<ContentItem> }, id?: string | number) => Promise<void>;
  itemToEdit?: ContentItem | null; // Optional prop for the item being edited
}

/**
 * Creates or edits an entry. The form starts from schema and itemToEdit when
 * it mounts: give it a new key each time it is opened for another entry.
 */
export function EntryFormModal({ isOpen, onClose, schema, itemToEdit, onSubmit }: EntryFormDrawerProps) {
  const [formData, setFormData] = useState<Record<string, unknown>>(() => initialValues(schema.attributes, itemToEdit));
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleChange = (key: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
    try {
      await onSubmit({ data: toEntryPayload(schema.attributes, formData) as Partial<ContentItem> }, itemToEdit?.id);
      onClose();
    } catch {
      // The parent shows the error.
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
          <EntryFields attributes={schema.attributes} values={formData} onChange={handleChange} idPrefix="entry-" />
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

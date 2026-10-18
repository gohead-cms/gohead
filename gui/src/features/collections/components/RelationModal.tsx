import React, { useState } from 'react';
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
  Button,
  FormControl,
  FormLabel,
  Select, 
  HStack,
  Text,
  FormHelperText, // NEW: Import FormHelperText
} from '@chakra-ui/react';
import { FaPlus } from 'react-icons/fa';

type RelationModalProps = {
  isOpen: boolean;
  onClose: () => void;
  sourceNodeLabel: string | null;
  targetNodeLabel: string | null;
  allNodeLabels: string[];
  onSave: (sourceId: string, targetId: string, relation: { type: string; label: string }) => void;
};

// Relation types accepted by the API.
type RelationType = 'oneToOne' | 'oneToMany' | 'manyToMany';

export function RelationModal({ isOpen, onClose, sourceNodeLabel, targetNodeLabel, allNodeLabels, onSave }: RelationModalProps) {
  const [relationType, setRelationType] = useState<RelationType>('oneToOne');
  const [relationLabel, setRelationLabel] = useState<string | null>(null);

  const handleSave = () => {
    if (sourceNodeLabel && targetNodeLabel && relationLabel) {
      onSave(sourceNodeLabel, targetNodeLabel, { type: relationType, label: relationLabel });
    }
  };

  const filteredNodeLabels = allNodeLabels.filter(label => label !== sourceNodeLabel && label !== targetNodeLabel);

  return (
    <Modal isOpen={isOpen} onClose={onClose}>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>Create Relation</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <HStack mb={4}>
            <Text fontWeight="bold">{sourceNodeLabel || 'Source'}</Text>
            <Text>→</Text>
            <Text fontWeight="bold">{targetNodeLabel || 'Target'}</Text>
          </HStack>

          <FormControl id="relationType" mb={4}>
            <FormLabel>Relation Type</FormLabel>
            <Select value={relationType} onChange={(e) => setRelationType(e.target.value as RelationType)}>
              <option value="oneToOne">One-to-One</option>
              <option value="oneToMany">One-to-Many</option>
              <option value="manyToMany">Many-to-Many</option>
            </Select>
          </FormControl>
          
          <FormControl id="relationLabel" mb={4}>
            <FormLabel>Relation Attribute Name</FormLabel>
            <Select
              placeholder="Select a collection"
              value={relationLabel || ''}
              onChange={(e) => setRelationLabel(e.target.value)}
            >
              {filteredNodeLabels.map((label) => (
                <option key={label} value={label}>
                  {label}
                </option>
              ))}
            </Select>
            <FormHelperText>This will be the name of the relation attribute on the source collection.</FormHelperText>
          </FormControl>
        </ModalBody>
        <ModalFooter>
          <Button variant="ghost" mr={3} onClick={onClose}>
            Cancel
          </Button>
          <Button
            colorScheme="teal"
            onClick={handleSave}
            leftIcon={<FaPlus />}
            isDisabled={!relationLabel}
          >
            Create
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}
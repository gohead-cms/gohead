import {
  Box,
  Checkbox,
  Flex,
  FormControl,
  FormErrorMessage,
  FormLabel,
  HStack,
  IconButton,
  Input,
  Select,
  Text,
} from "@chakra-ui/react";
import { FiArrowDown, FiArrowUp, FiTrash2 } from "react-icons/fi";
import { AttributeField, AttributeForm, supportsUnique, TYPE_HINTS, typeLabel } from "../schema";
import { AttributeOptions } from "./AttributeOptions";

interface Props {
  attr: AttributeForm;
  index: number;
  count: number;
  types: { value: string; label: string }[];
  errors: Partial<Record<AttributeField, string>>;
  collections: string[];
  components: string[];
  onChange: (patch: Partial<AttributeForm>) => void;
  onMove: (delta: -1 | 1) => void;
  onRemove: () => void;
}

/** One attribute: name, type, flags and type-specific options. */
export function AttributeEditor({ attr, index, count, types, errors, collections, components, onChange, onMove, onRemove }: Props) {
  const label = attr.name.trim() || `attribute ${index + 1}`;
  // A type the editor does not offer (e.g. "time") stays selectable.
  const typeOptions = types.some((t) => t.value === attr.type) ? types : [...types, { value: attr.type, label: typeLabel(attr.type) }];
  const unique = supportsUnique(attr);

  return (
    <Box role="group" aria-label={`Attribute ${index + 1}`} borderWidth="1px" borderRadius="md" p={4} bg="gray.50">
      <Flex gap={3} align="start" wrap={{ base: "wrap", lg: "nowrap" }}>
        <FormControl isInvalid={!!errors.name} isRequired flex="2" minW="180px">
          <FormLabel fontSize="sm">Attribute name</FormLabel>
          <Input size="sm" bg="white" value={attr.name} onChange={(e) => onChange({ name: e.target.value })} placeholder="title" />
          <FormErrorMessage>{errors.name}</FormErrorMessage>
        </FormControl>
        <FormControl isRequired flex="2" minW="180px">
          <FormLabel fontSize="sm">Type</FormLabel>
          <Select size="sm" bg="white" value={attr.type} onChange={(e) => onChange({ type: e.target.value })}>
            {typeOptions.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </Select>
        </FormControl>
        <HStack spacing={4} pt={{ base: 0, lg: 8 }} flexShrink={0}>
          <Checkbox isChecked={attr.required} onChange={(e) => onChange({ required: e.target.checked })}>
            Required
          </Checkbox>
          {unique && (
            <Checkbox isChecked={attr.unique} onChange={(e) => onChange({ unique: e.target.checked })}>
              Unique
            </Checkbox>
          )}
        </HStack>
        <HStack spacing={1} pt={{ base: 0, lg: 7 }} ml="auto" flexShrink={0}>
          <IconButton
            size="sm"
            variant="ghost"
            aria-label={`Move ${label} up`}
            icon={<FiArrowUp />}
            isDisabled={index === 0}
            onClick={() => onMove(-1)}
          />
          <IconButton
            size="sm"
            variant="ghost"
            aria-label={`Move ${label} down`}
            icon={<FiArrowDown />}
            isDisabled={index === count - 1}
            onClick={() => onMove(1)}
          />
          <IconButton size="sm" variant="ghost" colorScheme="red" aria-label={`Remove ${label}`} icon={<FiTrash2 />} onClick={onRemove} />
        </HStack>
      </Flex>
      {TYPE_HINTS[attr.type] && (
        <Text fontSize="xs" color="gray.500" mt={2}>
          {TYPE_HINTS[attr.type]}
        </Text>
      )}
      <Box mt={3}>
        <AttributeOptions attr={attr} errors={errors} onChange={onChange} collections={collections} components={components} />
      </Box>
    </Box>
  );
}

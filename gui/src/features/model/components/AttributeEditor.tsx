import { Box, Field, Flex, HStack, Icon, IconButton, Input, NativeSelect, Text } from "@chakra-ui/react";
import { Checkbox } from "@/components/ui/checkbox";
import { Tooltip } from "@/components/ui/tooltip";
import { FiArrowDown, FiArrowUp, FiGlobe, FiTrash2 } from "react-icons/fi";
import type { AttributeField, AttributeForm } from "../schema";
import { supportsLocalized, supportsUnique, TYPE_HINTS, typeLabel } from "../schema";
import { AttributeOptions } from "./AttributeOptions";

interface Props {
  attr: AttributeForm;
  index: number;
  count: number;
  types: { value: string; label: string }[];
  errors: Partial<Record<AttributeField, string>>;
  collections: string[];
  components: string[];
  /** The type can have localized attributes (collections and singletons). */
  localizable: boolean;
  onChange: (patch: Partial<AttributeForm>) => void;
  onMove: (delta: -1 | 1) => void;
  onRemove: () => void;
}

/** One attribute: name, type, flags and type-specific options. */
export function AttributeEditor({ attr, index, count, types, errors, collections, components, localizable, onChange, onMove, onRemove }: Props) {
  const label = attr.name.trim() || `attribute ${index + 1}`;
  // A type the editor does not offer (e.g. "time") stays selectable.
  const typeOptions = types.some((t) => t.value === attr.type) ? types : [...types, { value: attr.type, label: typeLabel(attr.type) }];
  const unique = supportsUnique(attr);

  return (
    <Box role="group" aria-label={`Attribute ${index + 1}`} borderWidth="1px" borderRadius="md" p={4} bg="gray.50">
      <Flex gap={3} align="start" wrap={{ base: "wrap", lg: "nowrap" }}>
        <Field.Root invalid={!!errors.name} required flex="2" minW="180px">
          <Field.Label fontSize="sm">
            Attribute name
            <Field.RequiredIndicator />
          </Field.Label>
          <Input size="xs" bg="white" value={attr.name} onChange={(e) => onChange({ name: e.target.value })} placeholder="title" />
          <Field.ErrorText>{errors.name}</Field.ErrorText>
        </Field.Root>
        <Field.Root required flex="2" minW="180px">
          <Field.Label fontSize="sm">
            Type
            <Field.RequiredIndicator />
          </Field.Label>
          <NativeSelect.Root size="xs">
            <NativeSelect.Field bg="white" value={attr.type} onChange={(e) => onChange({ type: e.target.value })}>
              {typeOptions.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </NativeSelect.Field>
            <NativeSelect.Indicator />
          </NativeSelect.Root>
        </Field.Root>
        <HStack gap={4} pt={{ base: 0, lg: 8 }} flexShrink={0}>
          <Checkbox size="sm" checked={attr.required} onCheckedChange={(e) => onChange({ required: !!e.checked })}>
            Required
          </Checkbox>
          {unique && (
            <Checkbox size="sm" checked={attr.unique} onCheckedChange={(e) => onChange({ unique: !!e.checked })}>
              Unique
            </Checkbox>
          )}
          {localizable && <LocalizedCheckbox attr={attr} onChange={onChange} />}
        </HStack>
        <HStack gap={1} pt={{ base: 0, lg: 7 }} ml="auto" flexShrink={0}>
          <IconButton size="xs" variant="ghost" aria-label={`Move ${label} up`} disabled={index === 0} onClick={() => onMove(-1)}>
            <FiArrowUp />
          </IconButton>
          <IconButton size="xs" variant="ghost" aria-label={`Move ${label} down`} disabled={index === count - 1} onClick={() => onMove(1)}>
            <FiArrowDown />
          </IconButton>
          <IconButton size="xs" variant="ghost" colorPalette="red" aria-label={`Remove ${label}`} onClick={onRemove}>
            <FiTrash2 />
          </IconButton>
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

/** "Localized": a value per locale. Relations are shared by every locale. */
function LocalizedCheckbox({ attr, onChange }: Pick<Props, "attr" | "onChange">) {
  const allowed = supportsLocalized(attr);
  const box = (
    <Checkbox
      size="sm"
      checked={allowed && attr.localized}
      disabled={!allowed}
      onCheckedChange={(e) => onChange({ localized: !!e.checked })}
    >
      <HStack as="span" gap={1}>
        <Icon as={FiGlobe} color={allowed && attr.localized ? "purple.500" : "gray.400"} />
        <span>Localized</span>
      </HStack>
    </Checkbox>
  );
  return (
    <Tooltip
      showArrow
      openDelay={300}
      content={
        allowed
          ? "A value per locale: translators edit it in each locale, other attributes are shared."
          : "Relations cannot be localized: links are shared by every locale."
      }
    >
      <span>{box}</span>
    </Tooltip>
  );
}

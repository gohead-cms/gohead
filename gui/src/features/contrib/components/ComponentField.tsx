import { useState } from "react";
import { Alert, AlertIcon, Box, Button, HStack, IconButton, Spinner, Text, Tooltip, VStack } from "@chakra-ui/react";
import { FiArrowDown, FiArrowUp, FiPlus, FiTrash2 } from "react-icons/fi";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import type { Attribute } from "../../../shared/types";
import { EntryFields } from "./EntryFields";
import { InvalidValue, attributeMap, initialValues, toEntryPayload } from "./entryValues";

interface ComponentSchema {
  name: string;
  description?: string;
  attributes: Attribute[] | Record<string, Attribute> | null;
}

/** The attributes of a component (GET /api/schema/components/{name}). */
function useComponentAttributes(name: string | undefined) {
  const path = name ? `/api/schema/components/${encodeURIComponent(name)}` : null;
  const { data, error } = useApiQuery(path, async () => {
    const json = await getJSON<ComponentSchema>(path!, `Could not load the component "${name}"`);
    return attributeMap(json.data?.attributes ?? undefined);
  });
  return { attributes: data, error };
}

interface Instance {
  /** Stable React key. */
  key: number;
  values: Record<string, unknown>;
}

let nextKey = 1;
const instance = (attributes: Record<string, Attribute>, value?: unknown): Instance => ({
  key: nextKey++,
  values: initialValues(attributes, value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : null),
});

/** The API value of one component instance, or an InvalidValue. */
function toValue(attributes: Record<string, Attribute>, values: Record<string, unknown>): unknown {
  try {
    return toEntryPayload(attributes, values);
  } catch (err) {
    return new InvalidValue(err instanceof Error ? err.message : "invalid value");
  }
}

interface ComponentFieldProps {
  id: string;
  attr: Attribute;
  value: unknown;
  onChange: (value: unknown) => void;
}

/**
 * Edits a component attribute: its fields, once (or a list of instances
 * for repeatable components). The form gets the API value.
 */
export function ComponentField(props: ComponentFieldProps) {
  const { attributes, error } = useComponentAttributes(props.attr.component);
  if (error) {
    return (
      <Alert status="error" borderRadius="md">
        <AlertIcon />
        {error}
      </Alert>
    );
  }
  if (!attributes) return <Spinner size="sm" />;
  // Mounted once the schema is known: its state starts from the value.
  return <ComponentEditor {...props} attributes={attributes} />;
}

function ComponentEditor({ id, attr, value, onChange, attributes }: ComponentFieldProps & { attributes: Record<string, Attribute> }) {
  const repeatable = !!attr.repeatable;
  const [items, setItems] = useState<Instance[]>(() => {
    if (repeatable) return (Array.isArray(value) ? value : []).map((v) => instance(attributes, v));
    return value && typeof value === "object" ? [instance(attributes, value)] : [];
  });

  const emit = (next: Instance[]) => {
    setItems(next);
    const values = next.map((i) => toValue(attributes, i.values));
    const invalid = values.find((v) => v instanceof InvalidValue);
    if (invalid) onChange(invalid);
    else onChange(repeatable ? values : (values[0] ?? null));
  };
  const update = (key: number, field: string, v: unknown) =>
    emit(items.map((i) => (i.key === key ? { ...i, values: { ...i.values, [field]: v } } : i)));
  const add = () => emit([...items, instance(attributes)]);
  const remove = (key: number) => emit(items.filter((i) => i.key !== key));
  const move = (index: number, by: number) => {
    const next = [...items];
    const [it] = next.splice(index, 1);
    next.splice(index + by, 0, it);
    emit(next);
  };

  const label = attr.component ?? "component";
  return (
    <VStack align="stretch" spacing={3} id={id}>
      {items.map((it, index) => (
        <Box key={it.key} borderWidth="1px" borderRadius="md" p={4} bg="gray.50">
          <HStack mb={3}>
            <Text fontSize="sm" fontWeight="semibold" color="gray.600" flex="1">
              {repeatable ? `${label} ${index + 1}` : label}
            </Text>
            {repeatable && (
              <>
                <Tooltip label="Move up">
                  <IconButton aria-label="Move up" icon={<FiArrowUp />} size="xs" variant="ghost" isDisabled={index === 0} onClick={() => move(index, -1)} />
                </Tooltip>
                <Tooltip label="Move down">
                  <IconButton
                    aria-label="Move down"
                    icon={<FiArrowDown />}
                    size="xs"
                    variant="ghost"
                    isDisabled={index === items.length - 1}
                    onClick={() => move(index, 1)}
                  />
                </Tooltip>
              </>
            )}
            <Tooltip label={repeatable ? "Remove" : "Clear"}>
              <IconButton aria-label={`Remove ${label}${repeatable ? ` ${index + 1}` : ""}`} icon={<FiTrash2 />} size="xs" variant="ghost" onClick={() => remove(it.key)} />
            </Tooltip>
          </HStack>
          <EntryFields attributes={attributes} values={it.values} onChange={(field, v) => update(it.key, field, v)} idPrefix={`${id}-${it.key}-`} />
        </Box>
      ))}
      {(repeatable || items.length === 0) && (
        <Button size="sm" variant="outline" leftIcon={<FiPlus />} onClick={add} alignSelf="flex-start">
          Add {label}
        </Button>
      )}
    </VStack>
  );
}

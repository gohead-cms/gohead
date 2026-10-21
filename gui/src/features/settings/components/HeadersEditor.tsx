import { Button, Field, HStack, IconButton, Input, Text, VStack } from "@chakra-ui/react";
import { FiPlus, FiX } from "react-icons/fi";
import { newHeaderRow } from "./settingsFormat";
import type { HeaderRow } from "./types";

/** Editable list of custom header name/value pairs. */
export function HeadersEditor({ rows, onChange }: { rows: HeaderRow[]; onChange: (rows: HeaderRow[]) => void }) {
  const update = (key: string, patch: Partial<HeaderRow>) => onChange(rows.map((r) => (r.key === key ? { ...r, ...patch } : r)));
  return (
    <VStack align="stretch" gap={2}>
      {rows.length > 0 && (
        <HStack aria-hidden fontSize="xs" color="gray.500" pr={10}>
          <Text flex="2">Name</Text>
          <Text flex="3">Value</Text>
        </HStack>
      )}
      {rows.map((r, i) => (
        <HStack key={r.key}>
          <Field.Root flex="2">
            <Field.Label srOnly>Header name {i + 1}</Field.Label>
            <Input size="xs" value={r.name} placeholder="Authorization" onChange={(e) => update(r.key, { name: e.target.value })} />
          </Field.Root>
          <Field.Root flex="3">
            <Field.Label srOnly>Header value {i + 1}</Field.Label>
            <Input size="xs" value={r.value} placeholder="Bearer …" onChange={(e) => update(r.key, { value: e.target.value })} />
          </Field.Root>
          <IconButton
            aria-label={`Remove header ${i + 1}`}
            size="xs"
            variant="ghost"
            onClick={() => onChange(rows.filter((x) => x.key !== r.key))}
          >
            <FiX />
          </IconButton>
        </HStack>
      ))}
      <Button size="xs" variant="outline" alignSelf="start" onClick={() => onChange([...rows, newHeaderRow()])}>
        <FiPlus />
        Add header
      </Button>
    </VStack>
  );
}

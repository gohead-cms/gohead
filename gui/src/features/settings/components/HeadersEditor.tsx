import { Button, FormControl, FormLabel, HStack, IconButton, Input, Text, VStack } from "@chakra-ui/react";
import { FiPlus, FiX } from "react-icons/fi";
import { newHeaderRow } from "./settingsFormat";
import type { HeaderRow } from "./types";

/** Editable list of custom header name/value pairs. */
export function HeadersEditor({ rows, onChange }: { rows: HeaderRow[]; onChange: (rows: HeaderRow[]) => void }) {
  const update = (key: string, patch: Partial<HeaderRow>) => onChange(rows.map((r) => (r.key === key ? { ...r, ...patch } : r)));
  return (
    <VStack align="stretch" spacing={2}>
      {rows.length > 0 && (
        <HStack aria-hidden fontSize="xs" color="gray.500" pr={10}>
          <Text flex="2">Name</Text>
          <Text flex="3">Value</Text>
        </HStack>
      )}
      {rows.map((r, i) => (
        <HStack key={r.key}>
          <FormControl flex="2">
            <FormLabel srOnly>
              Header name {i + 1}
            </FormLabel>
            <Input size="sm" value={r.name} placeholder="Authorization" onChange={(e) => update(r.key, { name: e.target.value })} />
          </FormControl>
          <FormControl flex="3">
            <FormLabel srOnly>
              Header value {i + 1}
            </FormLabel>
            <Input size="sm" value={r.value} placeholder="Bearer …" onChange={(e) => update(r.key, { value: e.target.value })} />
          </FormControl>
          <IconButton
            aria-label={`Remove header ${i + 1}`}
            icon={<FiX />}
            size="sm"
            variant="ghost"
            onClick={() => onChange(rows.filter((x) => x.key !== r.key))}
          />
        </HStack>
      ))}
      <Button size="sm" variant="outline" leftIcon={<FiPlus />} alignSelf="start" onClick={() => onChange([...rows, newHeaderRow()])}>
        Add header
      </Button>
    </VStack>
  );
}

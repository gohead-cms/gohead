import {
  Box,
  Button,
  CheckboxGroup,
  Field as ChakraField,
  Fieldset,
  HStack,
  IconButton,
  Input,
  NativeSelect,
  SimpleGrid,
  Text,
  Textarea,
  VStack,
  Wrap,
  WrapItem,
} from "@chakra-ui/react";
import { Checkbox } from "@/components/ui/checkbox";
import { FiPlus, FiTrash2 } from "react-icons/fi";
import { type AgentFunctionInfo } from "../../../shared/types";
import { type CollectionChoice } from "../hook/useEditorData";
import { type DraftErrors, type FunctionDraft, applyCatalog, emptyFunction, functionWarning } from "./agentForm";
import { EditorSection, Field, Notice } from "./EditorParts";

interface FunctionsEditorProps {
  functions: FunctionDraft[];
  onChange: (functions: FunctionDraft[]) => void;
  catalog: AgentFunctionInfo[];
  catalogError: string | null;
  collections: CollectionChoice[];
  errors: DraftErrors;
}

/** Catalog entries grouped by the prefix of their impl_key ("collections", "llm"…). */
function groupCatalog(catalog: AgentFunctionInfo[]): [string, AgentFunctionInfo[]][] {
  const groups = new Map<string, AgentFunctionInfo[]>();
  for (const info of catalog) {
    const prefix = info.impl_key.split(".")[0];
    groups.set(prefix, [...(groups.get(prefix) ?? []), info]);
  }
  return [...groups.entries()];
}

function FunctionCard({
  fn,
  index,
  catalog,
  collections,
  errors,
  onChange,
  onRemove,
}: {
  fn: FunctionDraft;
  index: number;
  catalog: AgentFunctionInfo[];
  collections: CollectionChoice[];
  errors: DraftErrors;
  onChange: (fn: FunctionDraft) => void;
  onRemove: () => void;
}) {
  const info = catalog.find((c) => c.impl_key === fn.impl_key);
  const scoped = info?.scoped ?? fn.impl_key.startsWith("collections.");
  const known = !fn.impl_key || Boolean(info);
  const warning = functionWarning(fn);
  const err = (field: string) => errors[`fn.${fn.uid}.${field}`];
  const label = fn.name || `function ${index + 1}`;
  // Scoped collections that no longer exist stay listed so they can be unchecked.
  const scopeChoices = [...new Set([...collections.map((c) => c.name), ...fn.collections])];

  return (
    <Box borderWidth="1px" borderRadius="md" p={4} bg="gray.50">
      <HStack justify="space-between" mb={3}>
        <Text fontWeight="semibold" fontSize="sm">
          {fn.name || "New function"}
          {fn.impl_key && (
            <Text as="span" color="gray.500" fontWeight="normal" ml={2}>
              → {fn.impl_key}
            </Text>
          )}
        </Text>
        <IconButton aria-label={`Remove ${label}`} size="xs" variant="ghost" colorPalette="red" onClick={onRemove}>
          <FiTrash2 />
        </IconButton>
      </HStack>
      <VStack gap={4} align="stretch">
        <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
          <Field label="Implementation" error={err("impl_key")} help={info?.description} isRequired>
            <NativeSelect.Root>
              <NativeSelect.Field
                bg="white"
                placeholder="Choose a built-in function"
                value={fn.impl_key}
                onChange={(e) => onChange(applyCatalog(fn, catalog.find((c) => c.impl_key === e.target.value), info))}
              >
                {!known && <option value={fn.impl_key}>{fn.impl_key} (unknown)</option>}
                {groupCatalog(catalog).map(([group, items]) => (
                  <optgroup key={group} label={group}>
                    {items.map((c) => (
                      <option key={c.impl_key} value={c.impl_key}>
                        {c.impl_key}
                      </option>
                    ))}
                  </optgroup>
                ))}
              </NativeSelect.Field>
              <NativeSelect.Indicator />
            </NativeSelect.Root>
          </Field>
          <Field label="Function name (seen by the model)" error={err("name")} isRequired>
            <Input bg="white" value={fn.name} onChange={(e) => onChange({ ...fn, name: e.target.value })} placeholder="save_summary" />
          </Field>
        </SimpleGrid>
        <Field label="Description (for the model)" help="Tell the model when to call it.">
          <Input bg="white" value={fn.description} onChange={(e) => onChange({ ...fn, description: e.target.value })} />
        </Field>
        {scoped && (
          <Fieldset.Root size="sm">
            <Fieldset.Legend fontSize="sm">Allowed collections</Fieldset.Legend>
            {scopeChoices.length > 0 ? (
              <CheckboxGroup value={fn.collections} onValueChange={(v) => onChange({ ...fn, collections: v.map(String) })}>
                <Wrap columnGap={5} rowGap={1}>
                  {scopeChoices.map((name) => (
                    <WrapItem key={name}>
                      <Checkbox value={name} size="sm" bg="white">
                        {name}
                      </Checkbox>
                    </WrapItem>
                  ))}
                </Wrap>
              </CheckboxGroup>
            ) : (
              <Text fontSize="sm" color="gray.500">
                No collections yet.
              </Text>
            )}
            <Fieldset.HelperText>None checked: the function may use every collection.</Fieldset.HelperText>
          </Fieldset.Root>
        )}
        {warning && (
          <Notice status="warning" borderRadius="md" fontSize="sm">
            {warning}
          </Notice>
        )}
        <ChakraField.Root invalid={Boolean(err("parameters"))} required gap={1}>
          <HStack justify="space-between" w="full">
            <ChakraField.Label fontSize="sm">
              Parameters (JSON schema)
              <ChakraField.RequiredIndicator />
            </ChakraField.Label>
            {info && (
              <Button
                size="2xs"
                variant="plain"
                color="gray.500"
                px={0}
                h="auto"
                _hover={{ textDecoration: "underline" }}
                onClick={() => onChange({ ...fn, parameters: JSON.stringify(info.parameters ?? {}, null, 2) })}
              >
                Reset to default
              </Button>
            )}
          </HStack>
          <Textarea
            bg="white"
            fontFamily="mono"
            fontSize="xs"
            rows={8}
            spellCheck={false}
            value={fn.parameters}
            onChange={(e) => onChange({ ...fn, parameters: e.target.value })}
          />
          {err("parameters") ? (
            <ChakraField.ErrorText fontSize="sm">{err("parameters")}</ChakraField.ErrorText>
          ) : (
            <ChakraField.HelperText fontSize="sm">Property names must match the arguments the implementation reads.</ChakraField.HelperText>
          )}
        </ChakraField.Root>
      </VStack>
    </Box>
  );
}

/** The agent's functions: each maps a name the model sees to a built-in implementation. */
export function FunctionsEditor({ functions, onChange, catalog, catalogError, collections, errors }: FunctionsEditorProps) {
  const add = () => onChange([...functions, emptyFunction()]);
  return (
    <EditorSection
      title={`Functions (${functions.length})`}
      description="Tools the model may call. Pick a built-in implementation; its description and parameters are filled in for you."
      actions={
        <Button size="xs" variant="subtle" onClick={add} flexShrink={0}>
          <FiPlus />
          Add function
        </Button>
      }
    >
      {catalogError && (
        <Notice status="error" mb={4} borderRadius="md">
          {catalogError}
        </Notice>
      )}
      {functions.length === 0 ? (
        <Text fontSize="sm" color="gray.500">
          No functions: the agent can only answer with text.{" "}
          <Button variant="plain" size="xs" color="gray.500" px={0} h="auto" verticalAlign="baseline" _hover={{ textDecoration: "underline" }} onClick={add}>
            Add one
          </Button>
        </Text>
      ) : (
        <VStack gap={4} align="stretch">
          {functions.map((fn, i) => (
            <FunctionCard
              key={fn.uid}
              fn={fn}
              index={i}
              catalog={catalog}
              collections={collections}
              errors={errors}
              onChange={(next) => onChange(functions.map((f) => (f.uid === fn.uid ? next : f)))}
              onRemove={() => onChange(functions.filter((f) => f.uid !== fn.uid))}
            />
          ))}
        </VStack>
      )}
    </EditorSection>
  );
}

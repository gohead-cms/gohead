import { useState } from "react";
import {
  Badge,
  Box,
  Button,
  CheckboxGroup,
  Code,
  Collapsible,
  Fieldset,
  HStack,
  Icon,
  IconButton,
  Input,
  SimpleGrid,
  Text,
  Textarea,
  VStack,
  Wrap,
  WrapItem,
  chakra,
} from "@chakra-ui/react";
import { FiChevronDown, FiChevronRight, FiCpu, FiDatabase, FiPlus, FiTerminal, FiTrash2 } from "react-icons/fi";
import { Checkbox } from "@/components/ui/checkbox";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { type AgentFunctionInfo } from "../../../shared/types";
import { type CollectionChoice } from "../hook/useEditorData";
import { type DraftErrors, type ToolDraft, newTool, toolInfo, toolName, toolWarning } from "./agentForm";
import { EditorSection, Field, Notice } from "./EditorParts";

const GROUPS: { id: string; title: string; help: string; icon: typeof FiDatabase }[] = [
  { id: "content", title: "Content", help: "Read and change items. Choose the collections each tool may use.", icon: FiDatabase },
  { id: "ai", title: "AI helpers", help: "Ask a model about a text: summary, tags, moderation… They use the instance's default model.", icon: FiCpu },
  { id: "system", title: "System", help: "", icon: FiTerminal },
];

const groupOf = (info: AgentFunctionInfo | undefined) => GROUPS.find((g) => g.id === info?.group) ?? GROUPS[2];

/** The catalog, by group: click a tool to add it. */
function ToolPicker({
  open,
  onClose,
  catalog,
  onPick,
}: {
  open: boolean;
  onClose: () => void;
  catalog: AgentFunctionInfo[];
  onPick: (info: AgentFunctionInfo) => void;
}) {
  return (
    <DialogRoot open={open} onOpenChange={(e) => !e.open && onClose()} size="xl" scrollBehavior="inside">
      <DialogContent maxW="3xl">
        <DialogHeader>
          <DialogTitle>Add a tool</DialogTitle>
        </DialogHeader>
        <DialogBody pb={6}>
          <VStack align="stretch" gap={6}>
            {GROUPS.map((g) => {
              const items = catalog.filter((c) => groupOf(c).id === g.id);
              if (items.length === 0) return null;
              return (
                <Box key={g.id} as="section" aria-label={g.title}>
                  <HStack gap={2} mb={1}>
                    <Icon as={g.icon} color="purple.500" />
                    <Text fontWeight="bold">{g.title}</Text>
                  </HStack>
                  {g.help && (
                    <Text fontSize="sm" color="fg.muted" mb={3}>
                      {g.help}
                    </Text>
                  )}
                  <SimpleGrid columns={{ base: 1, md: 2 }} gap={2}>
                    {items.map((info) => (
                      <chakra.button
                        key={info.impl_key}
                        type="button"
                        textAlign="start"
                        borderWidth="1px"
                        borderRadius="md"
                        p={3}
                        bg="white"
                        _hover={{ borderColor: "purple.400", bg: "purple.50" }}
                        onClick={() => onPick(info)}
                      >
                        <HStack justify="space-between" align="start" gap={2}>
                          <Text fontWeight="semibold" fontSize="sm">
                            {info.label}
                          </Text>
                          {info.writes && (
                            <Badge colorPalette="orange" size="sm" flexShrink={0}>
                              Writes
                            </Badge>
                          )}
                        </HStack>
                        <Text fontSize="sm" color="fg.muted" mt={1}>
                          {info.description}
                        </Text>
                        <Code fontSize="xs" mt={2} bg="transparent" px={0} color="gray.500">
                          {info.use}
                        </Code>
                      </chakra.button>
                    ))}
                  </SimpleGrid>
                </Box>
              );
            })}
          </VStack>
        </DialogBody>
        <DialogCloseTrigger />
      </DialogContent>
    </DialogRoot>
  );
}

function ToolCard({
  tool,
  catalog,
  collections,
  errors,
  onChange,
  onRemove,
}: {
  tool: ToolDraft;
  catalog: AgentFunctionInfo[];
  collections: CollectionChoice[];
  errors: DraftErrors;
  onChange: (t: ToolDraft) => void;
  onRemove: () => void;
}) {
  const info = toolInfo(catalog, tool.use);
  const group = groupOf(info);
  const label = info?.label ?? tool.use;
  const warning = toolWarning(tool, catalog);
  const err = (field: string) => errors[`tool.${tool.uid}.${field}`];
  const customized = Boolean(tool.name.trim() || tool.description.trim() || tool.parameters.trim());
  const [open, setOpen] = useState(Boolean(err("name") || err("parameters")));
  // Collections that no longer exist stay listed so they can be unchecked.
  const scopeChoices = [...new Set([...collections.map((c) => c.name), ...tool.on])];
  const scoped = info?.scoped ?? tool.use.startsWith("collections.");

  return (
    <Box borderWidth="1px" borderRadius="md" p={4} bg="gray.50" data-testid={`tool-${tool.use}`}>
      <HStack justify="space-between" align="start" gap={3}>
        <HStack align="start" gap={3} minW={0}>
          <Icon as={group.icon} color="purple.500" mt={1} flexShrink={0} />
          <Box minW={0}>
            <HStack gap={2} wrap="wrap">
              <Text fontWeight="semibold">{label}</Text>
              {info?.writes && (
                <Badge colorPalette="orange" size="sm">
                  Writes
                </Badge>
              )}
              {!info && (
                <Badge colorPalette="red" size="sm">
                  Not a built-in tool
                </Badge>
              )}
            </HStack>
            <Text fontSize="sm" color="fg.muted">
              {tool.description.trim() || info?.description}
            </Text>
            <Text fontSize="xs" color="gray.500" mt={1}>
              <Code fontSize="xs" bg="transparent" px={0}>
                {tool.use}
              </Code>
              , called <Code fontSize="xs">{toolName(tool, catalog)}</Code> by the model
            </Text>
          </Box>
        </HStack>
        <IconButton aria-label={`Remove ${label}`} size="xs" variant="ghost" colorPalette="red" onClick={onRemove}>
          <FiTrash2 />
        </IconButton>
      </HStack>

      <VStack gap={3} align="stretch" mt={3} ps={7}>
        {scoped && (
          <Fieldset.Root size="sm">
            <Fieldset.Legend fontSize="sm">On collections</Fieldset.Legend>
            {scopeChoices.length > 0 ? (
              <CheckboxGroup value={tool.on} onValueChange={(v) => onChange({ ...tool, on: v.map(String) })}>
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
            <Fieldset.HelperText>None checked: every collection.</Fieldset.HelperText>
          </Fieldset.Root>
        )}
        {warning && (
          <Notice status="warning" borderRadius="md" fontSize="sm">
            {warning}
          </Notice>
        )}
        {err("name") && !open && (
          <Notice status="error" borderRadius="md" fontSize="sm">
            {err("name")}
          </Notice>
        )}

        <Collapsible.Root open={open} onOpenChange={(e) => setOpen(e.open)}>
          <Collapsible.Trigger asChild>
            <Button size="xs" variant="plain" color="gray.600" px={0} h="auto" _hover={{ textDecoration: "underline" }}>
              {open ? <FiChevronDown /> : <FiChevronRight />}
              Customize{customized ? " (customized)" : ""}
            </Button>
          </Collapsible.Trigger>
          <Collapsible.Content>
            <VStack gap={3} align="stretch" mt={3}>
              <Field label="Tool name (seen by the model)" error={err("name")} help="Empty: the default. Use it when the prompt refers to the tool, or for two tools of the same kind.">
                <Input
                  bg="white"
                  value={tool.name}
                  placeholder={info?.name ?? ""}
                  onChange={(e) => onChange({ ...tool, name: e.target.value })}
                />
              </Field>
              <Field label="What the model is told" help="Empty: the tool's own description.">
                <Textarea
                  bg="white"
                  rows={2}
                  value={tool.description}
                  placeholder={info?.description ?? ""}
                  onChange={(e) => onChange({ ...tool, description: e.target.value })}
                />
              </Field>
              <Field
                label="Arguments (JSON schema)"
                error={err("parameters")}
                help="Empty: the tool's own arguments. Property names must be ones the tool reads."
              >
                <Textarea
                  bg="white"
                  fontFamily="mono"
                  fontSize="xs"
                  rows={tool.parameters ? 8 : 2}
                  spellCheck={false}
                  value={tool.parameters}
                  placeholder="Leave empty to use the tool's own arguments"
                  onChange={(e) => onChange({ ...tool, parameters: e.target.value })}
                />
              </Field>
              {info && !tool.parameters && (
                <Button
                  size="2xs"
                  variant="plain"
                  color="gray.500"
                  px={0}
                  h="auto"
                  alignSelf="start"
                  _hover={{ textDecoration: "underline" }}
                  onClick={() => onChange({ ...tool, parameters: JSON.stringify(info.parameters ?? {}, null, 2) })}
                >
                  Start from the tool's arguments
                </Button>
              )}
            </VStack>
          </Collapsible.Content>
        </Collapsible.Root>
      </VStack>
    </Box>
  );
}

/** The agent's tools: picked from the catalog, scoped to collections, optionally customized. */
export function ToolsEditor({
  tools,
  onChange,
  catalog,
  catalogError,
  collections,
  triggerCollection,
  errors,
}: {
  tools: ToolDraft[];
  onChange: (tools: ToolDraft[]) => void;
  catalog: AgentFunctionInfo[];
  catalogError: string | null;
  collections: CollectionChoice[];
  /** The trigger's collection: new content tools start scoped to it. */
  triggerCollection: string;
  errors: DraftErrors;
}) {
  const [picking, setPicking] = useState(false);
  const add = (info: AgentFunctionInfo) => {
    onChange([...tools, newTool(info, triggerCollection)]);
    setPicking(false);
  };
  return (
    <EditorSection
      title={`Tools (${tools.length})`}
      description="What the agent may do. Pick tools from the catalog: the model is told what each one does and which arguments it takes."
      actions={
        <Button size="xs" variant="subtle" onClick={() => setPicking(true)} flexShrink={0} disabled={catalog.length === 0}>
          <FiPlus />
          Add a tool
        </Button>
      }
    >
      {catalogError && (
        <Notice status="error" mb={4} borderRadius="md">
          {catalogError}
        </Notice>
      )}
      {tools.length === 0 ? (
        <Text fontSize="sm" color="gray.500">
          No tools: the agent can only answer with text.{" "}
          <Button
            variant="plain"
            size="xs"
            color="gray.500"
            px={0}
            h="auto"
            verticalAlign="baseline"
            _hover={{ textDecoration: "underline" }}
            onClick={() => setPicking(true)}
            disabled={catalog.length === 0}
          >
            Add one
          </Button>
        </Text>
      ) : (
        <VStack gap={3} align="stretch">
          {tools.map((t) => (
            <ToolCard
              key={t.uid}
              tool={t}
              catalog={catalog}
              collections={collections}
              errors={errors}
              onChange={(next) => onChange(tools.map((x) => (x.uid === t.uid ? next : x)))}
              onRemove={() => onChange(tools.filter((x) => x.uid !== t.uid))}
            />
          ))}
        </VStack>
      )}
      <ToolPicker open={picking} onClose={() => setPicking(false)} catalog={catalog} onPick={add} />
    </EditorSection>
  );
}

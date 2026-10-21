import {
  Box,
  Button,
  CheckboxGroup,
  Code,
  Field as ChakraField,
  Fieldset,
  HStack,
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
import { InputGroup } from "@/components/ui/input-group";
import { Switch } from "@/components/ui/switch";
import { API_BASE_URL } from "../../../services/api";
import { type CollectionChoice } from "../hook/useEditorData";
import { type AgentDraft, type DraftErrors, EVENTS, LIMIT_FIELDS, PROVIDERS, TRIGGER_TYPES, generateToken } from "./agentForm";
import { CopyButton, EditorSection, Field, Notice, RequiredMark } from "./EditorParts";

export interface SectionProps {
  draft: AgentDraft;
  set: <K extends keyof AgentDraft>(key: K, value: AgentDraft[K]) => void;
  errors: DraftErrors;
}

const MODEL_HINTS: Record<string, string> = {
  openai: "gpt-4o-mini",
  anthropic: "claude-sonnet-4-5",
  ollama: "llama3.1",
};

const KEY_ENV: Record<string, string> = {
  openai: "OPENAI_API_KEY",
  anthropic: "ANTHROPIC_API_KEY",
};

export function BasicsSection({ draft, set, errors }: SectionProps) {
  return (
    <EditorSection title="Basics">
      <VStack gap={4} align="stretch">
        <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
          <Field label="Name" error={errors.name} isRequired help="Unique; used in URLs and the run history.">
            <Input value={draft.name} onChange={(e) => set("name", e.target.value)} placeholder="summarizer" autoComplete="off" />
          </Field>
          <Field label="Max turns" error={errors.max_turns} isRequired help="Model round-trips per run; each function call uses one. Start with 3–5.">
            <Input type="number" min={1} value={draft.max_turns} onChange={(e) => set("max_turns", e.target.value)} />
          </Field>
        </SimpleGrid>
        <Field
          label="System prompt"
          error={errors.system_prompt}
          isRequired
          help="Be explicit about which function to call and when, with exact collection and field names."
        >
          <Textarea rows={8} value={draft.system_prompt} onChange={(e) => set("system_prompt", e.target.value)} />
        </Field>
      </VStack>
    </EditorSection>
  );
}

function ApiKeyField({ draft, set }: SectionProps) {
  const env = KEY_ENV[draft.provider];
  let help: React.ReactNode;
  if (draft.api_key) help = draft.api_key_stored ? "Replaces the stored key when you save." : "Stored with the agent and masked in responses.";
  else if (draft.api_key_stored)
    help = (
      <>
        A key is stored and kept unchanged.{" "}
        <Button variant="plain" size="2xs" colorPalette="red" color="red.500" px={0} h="auto" _hover={{ textDecoration: "underline" }} onClick={() => set("api_key_stored", false)}>
          Remove stored key
        </Button>
      </>
    );
  else help = env ? `Empty: the instance default (llm.api_key or ${env}).` : "Empty: the instance default.";
  return (
    <Field label="API key" help={help}>
      <Input
        type="password"
        autoComplete="new-password"
        value={draft.api_key}
        onChange={(e) => set("api_key", e.target.value)}
        placeholder={draft.api_key_stored ? "Unchanged (stored)" : "sk-…"}
      />
    </Field>
  );
}

export function ModelSection(props: SectionProps) {
  const { draft, set, errors } = props;
  return (
    <EditorSection title="Model" description="Empty fields use the instance defaults (the llm block of the configuration) when the provider matches.">
      <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
        <Field label="Provider" error={errors.provider} isRequired>
          <NativeSelect.Root>
            <NativeSelect.Field value={draft.provider} onChange={(e) => set("provider", e.target.value)}>
              {PROVIDERS.map((p) => (
                <option key={p} value={p}>
                  {p}
                </option>
              ))}
            </NativeSelect.Field>
            <NativeSelect.Indicator />
          </NativeSelect.Root>
        </Field>
        <Field label="Model" error={errors.model} isRequired={draft.provider === "ollama"} help={draft.provider === "ollama" ? "A model that supports tool calling." : "Empty: the default model."}>
          <Input value={draft.model} onChange={(e) => set("model", e.target.value)} placeholder={MODEL_HINTS[draft.provider]} />
        </Field>
        {draft.provider !== "ollama" && <ApiKeyField {...props} />}
        <Field
          label="Base URL"
          error={errors.base_url}
          help={draft.provider === "ollama" ? "Default http://localhost:11434." : "Optional: an OpenAI-compatible endpoint or a proxy."}
        >
          <Input value={draft.base_url} onChange={(e) => set("base_url", e.target.value)} placeholder="https://…" />
        </Field>
        <Field label="Input price (USD per million tokens)" error={errors.input_cost} help="Used to compute costs and the cost limit.">
          <Input type="number" min={0} step="any" value={draft.input_cost} onChange={(e) => set("input_cost", e.target.value)} placeholder="0.15" />
        </Field>
        <Field label="Output price (USD per million tokens)" error={errors.output_cost}>
          <Input type="number" min={0} step="any" value={draft.output_cost} onChange={(e) => set("output_cost", e.target.value)} placeholder="0.60" />
        </Field>
      </SimpleGrid>
    </EditorSection>
  );
}

function WebhookFields({ draft, set, errors, agentId }: SectionProps & { agentId?: number }) {
  const url = agentId ? `${API_BASE_URL}/agents/webhook/${agentId}` : null;
  return (
    <VStack gap={4} align="stretch">
      <ChakraField.Root gap={1}>
        <ChakraField.Label fontSize="sm">Trigger URL</ChakraField.Label>
        {url ? (
          <HStack>
            <Code px={2} py={1} fontSize="sm" wordBreak="break-all">
              POST {url}
            </Code>
            <CopyButton value={url} />
          </HStack>
        ) : (
          <Text fontSize="sm" color="gray.600">
            Shown once the agent is saved: <Code fontSize="xs">POST {API_BASE_URL}/agents/webhook/&#123;id&#125;</Code>
          </Text>
        )}
        <ChakraField.HelperText fontSize="sm">
          Send a JSON body with the header <Code fontSize="xs">Webhook-Token</Code>; the body is passed to the agent.
        </ChakraField.HelperText>
      </ChakraField.Root>
      <ChakraField.Root invalid={Boolean(errors.webhook_token)} gap={1}>
        <ChakraField.Label fontSize="sm">Webhook token</ChakraField.Label>
        {draft.webhook_token ? (
          <>
            <HStack w="full">
              <Input value={draft.webhook_token} readOnly fontFamily="mono" fontSize="sm" aria-label="New webhook token" />
              <CopyButton value={draft.webhook_token} />
              <Button size="xs" variant="ghost" onClick={() => set("webhook_token", generateToken())} flexShrink={0}>
                Regenerate
              </Button>
            </HStack>
            <Notice status="info" mt={1} borderRadius="md" fontSize="sm">
              Copy this token now: after saving it is stored and only shown masked.
              {draft.webhook_token_stored && " Saving replaces the current token."}
            </Notice>
          </>
        ) : (
          <HStack>
            <Text fontSize="sm" color="gray.600">
              A token is stored (hidden) and kept unchanged.
            </Text>
            <Button size="xs" variant="outline" onClick={() => set("webhook_token", generateToken())}>
              Generate a new token
            </Button>
          </HStack>
        )}
        <ChakraField.ErrorText fontSize="sm">{errors.webhook_token}</ChakraField.ErrorText>
      </ChakraField.Root>
    </VStack>
  );
}

export function TriggerSection(props: SectionProps & { collections: CollectionChoice[]; collectionsError: string | null; agentId?: number }) {
  const { draft, set, errors, collections, collectionsError } = props;
  const names = collections.map((c) => c.name);
  const chosen = collections.find((c) => c.name === draft.collection);
  const draftOnlyPicked = chosen && !chosen.draftAndPublish && draft.events.some((ev) => EVENTS.find((x) => x.value === ev)?.draftOnly);
  return (
    <EditorSection title="Trigger" description="What starts a run. Any agent can also be run by hand from the agents page.">
      <VStack gap={4} align="stretch">
        <Field label="Trigger type" isRequired>
          <NativeSelect.Root maxW={{ md: "sm" }}>
            <NativeSelect.Field value={draft.trigger_type} onChange={(e) => set("trigger_type", e.target.value)}>
              {TRIGGER_TYPES.map((t) => (
                <option key={t.value} value={t.value}>
                  {t.label}
                </option>
              ))}
            </NativeSelect.Field>
            <NativeSelect.Indicator />
          </NativeSelect.Root>
        </Field>

        {draft.trigger_type === "collection_event" && (
          <>
            {collectionsError && (
              <Notice status="error" borderRadius="md">
                {collectionsError}
              </Notice>
            )}
            <Field label="Collection" error={errors.collection} isRequired>
              <NativeSelect.Root maxW={{ md: "sm" }}>
                <NativeSelect.Field placeholder="Choose a collection" value={draft.collection} onChange={(e) => set("collection", e.target.value)}>
                  {draft.collection && !names.includes(draft.collection) && <option value={draft.collection}>{draft.collection} (missing)</option>}
                  {names.map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </NativeSelect.Field>
                <NativeSelect.Indicator />
              </NativeSelect.Root>
            </Field>
            <Fieldset.Root invalid={Boolean(errors.events)} size="sm">
              <Fieldset.Legend fontSize="sm">
                Events
                <RequiredMark />
              </Fieldset.Legend>
              <CheckboxGroup value={draft.events} onValueChange={(v) => set("events", v.map(String))}>
                <Wrap columnGap={5} rowGap={1}>
                  {EVENTS.map((ev) => (
                    <WrapItem key={ev.value}>
                      <Checkbox value={ev.value} size="sm">
                        {ev.label}
                        <Text as="span" color="gray.500" fontSize="xs" ml={1}>
                          {ev.value}
                        </Text>
                      </Checkbox>
                    </WrapItem>
                  ))}
                </Wrap>
              </CheckboxGroup>
              {errors.events ? (
                <Fieldset.ErrorText>{errors.events}</Fieldset.ErrorText>
              ) : (
                <Fieldset.HelperText>Published and unpublished only happen in draft-and-publish collections. Writes made by agents emit no events.</Fieldset.HelperText>
              )}
            </Fieldset.Root>
            {draftOnlyPicked && (
              <Notice status="warning" borderRadius="md" fontSize="sm">
                '{draft.collection}' does not use draft-and-publish, so published / unpublished events never fire.
              </Notice>
            )}
          </>
        )}

        {draft.trigger_type === "cron" && (
          <Field label="Cron expression" error={errors.cron} isRequired help="Five fields, evaluated in UTC. e.g. 0 7 * * 1-5 runs on weekdays at 07:00 UTC.">
            <Input value={draft.cron} onChange={(e) => set("cron", e.target.value)} placeholder="0 7 * * 1-5" fontFamily="mono" maxW={{ md: "sm" }} />
          </Field>
        )}

        {draft.trigger_type === "webhook" && <WebhookFields {...props} />}
      </VStack>
    </EditorSection>
  );
}

export function LimitsSection({ draft, set, errors }: SectionProps) {
  const none = LIMIT_FIELDS.every((f) => !Number(draft.limits[f.key] || 0));
  const automatic = draft.trigger_type !== "manual";
  return (
    <EditorSection title="Limits" description="Empty or 0: no limit. Windows are rolling (last 60 minutes, last 24 hours). A run over a limit is skipped or stopped and recorded as limited.">
      {none && automatic && (
        <Notice status="warning" borderRadius="md" fontSize="sm" mb={4}>
          No limits: this agent can run and spend without bound. Set at least a number of runs per hour.
        </Notice>
      )}
      <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} gap={4}>
        {LIMIT_FIELDS.map((f) => (
          <Field key={f.key} label={f.label} error={errors[`limits.${f.key}`]} help={f.help}>
            <InputGroup w="full" startAddon={f.decimal ? "$" : undefined}>
              <Input
                type="number"
                min={0}
                step={f.decimal ? "any" : 1}
                value={draft.limits[f.key]}
                onChange={(e) => set("limits", { ...draft.limits, [f.key]: e.target.value })}
              />
            </InputGroup>
          </Field>
        ))}
      </SimpleGrid>
    </EditorSection>
  );
}

export function MemorySection({ draft, set, errors }: SectionProps) {
  return (
    <EditorSection title="Memory" description="How the conversation is stored. The last 40 messages are replayed on each run. The defaults suit most agents.">
      <SimpleGrid columns={{ base: 1, md: 2 }} gap={4}>
        <Field label="Memory type" error={errors.memory_type} isRequired>
          <Input value={draft.memory_type} onChange={(e) => set("memory_type", e.target.value)} placeholder="in-memory" />
        </Field>
        <Field label="Session scope" error={errors.session_scope} isRequired>
          <Input value={draft.session_scope} onChange={(e) => set("session_scope", e.target.value)} placeholder="user" />
        </Field>
      </SimpleGrid>
    </EditorSection>
  );
}

export function ApprovalSection({ draft, set }: SectionProps) {
  return (
    <EditorSection title="Review" description="Whether people check the agent's content changes before they are made.">
      <ChakraField.Root flexDirection="row" alignItems="start" gap={3}>
        <Switch
          size="sm"
          mt={1}
          colorPalette="teal"
          checked={draft.require_approval}
          onCheckedChange={(e) => set("require_approval", e.checked)}
        />
        <Box>
          <ChakraField.Label fontSize="md">Require approval</ChakraField.Label>
          <ChakraField.HelperText fontSize="sm" mt={1}>
            Creates, updates, translations and deletes are not applied: they wait on the Approvals page until an editor approves, edits or
            rejects them.
          </ChakraField.HelperText>
        </Box>
      </ChakraField.Root>
    </EditorSection>
  );
}

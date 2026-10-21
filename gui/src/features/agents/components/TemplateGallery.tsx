import { type FormEvent, useEffect, useRef, useState } from "react";
import { Box, Button, CheckboxGroup, Field, Fieldset, Heading, Icon, Input, NativeSelect, SimpleGrid, Spinner, Text, VStack, Wrap, WrapItem } from "@chakra-ui/react";
import { Checkbox } from "@/components/ui/checkbox";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { Switch } from "@/components/ui/switch";
import { type IconType } from "react-icons";
import { FiCpu, FiFileText, FiGlobe, FiPlus, FiShield, FiTag } from "react-icons/fi";
import { api } from "../../../services/api";
import { type AgentDefinition, type AgentTemplate, type AgentTemplateParam, type Attribute } from "../../../shared/types";
import { errorMessage } from "../../../shared/utils/errors";
import { type CollectionChoice, useAgentTemplates, useCollectionAttributes } from "../hook/useEditorData";
import { Notice, RequiredMark } from "./EditorParts";

const TEMPLATE_ICONS: Record<string, IconType> = {
  summarizer: FiFileText,
  tagger: FiTag,
  moderator: FiShield,
  translator: FiGlobe,
};

function GalleryCard({
  icon,
  title,
  description,
  action,
  onClick,
}: {
  icon: IconType;
  title: string;
  description: string;
  action: string;
  onClick: () => void;
}) {
  return (
    <Box
      as="article"
      aria-label={title}
      bg="white"
      borderWidth="1px"
      borderRadius="lg"
      p={5}
      display="flex"
      flexDirection="column"
      gap={3}
      _hover={{ borderColor: "teal.300", shadow: "sm" }}
      transition="all 0.15s"
    >
      <Icon as={icon} boxSize={6} color="teal.500" />
      <Box flex="1">
        <Heading size="md" fontWeight="bold" mb={1}>
          {title}
        </Heading>
        <Text fontSize="sm" color="gray.600">
          {description}
        </Text>
      </Box>
      <Button size="xs" colorPalette="teal" variant="outline" alignSelf="start" onClick={onClick}>
        {action}
      </Button>
    </Box>
  );
}

interface TemplateGalleryProps {
  collections: CollectionChoice[];
  onBlank: () => void;
  /** A template was rendered: review the definition in the form. */
  onRendered: (definition: AgentDefinition, template: AgentTemplate) => void;
}

/** The start of "New agent": a blank agent, or a ready-made one from a template. */
export function TemplateGallery({ collections, onBlank, onRendered }: TemplateGalleryProps) {
  const { templates, loading, error } = useAgentTemplates();
  const [chosen, setChosen] = useState<AgentTemplate | null>(null);

  return (
    <VStack align="stretch" gap={4}>
      <Box>
        <Heading size="xl" fontWeight="bold">
          Start from a template
        </Heading>
        <Text fontSize="sm" color="gray.500" mt={1}>
          A template writes the system prompt, trigger and functions for a common task. You review everything before the agent is created.
        </Text>
      </Box>
      {error && (
        <Notice status="warning" borderRadius="md">
          {error}
        </Notice>
      )}
      <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} gap={4}>
        <GalleryCard
          icon={FiPlus}
          title="Blank agent"
          description="Write the system prompt, choose the trigger and the functions yourself."
          action="Start blank"
          onClick={onBlank}
        />
        {templates.map((t) => (
          <GalleryCard
            key={t.id}
            icon={TEMPLATE_ICONS[t.id] ?? FiCpu}
            title={t.name}
            description={t.description}
            action="Use template"
            onClick={() => setChosen(t)}
          />
        ))}
      </SimpleGrid>
      {loading && <Spinner size="sm" />}
      {/* A new choice starts with a fresh form. */}
      {chosen && (
        <TemplateParamsModal
          key={chosen.id}
          template={chosen}
          collections={collections}
          onClose={() => setChosen(null)}
          onRendered={(def) => onRendered(def, chosen)}
        />
      )}
    </VStack>
  );
}

type Values = Record<string, string | string[] | boolean>;

/** Starting values: the defaults, as form values. */
function initialValues(params: AgentTemplateParam[]): Values {
  const v: Values = {};
  for (const p of params) {
    if (p.type === "bool") v[p.key] = p.default === true;
    else if (p.type === "fields") v[p.key] = Array.isArray(p.default) ? p.default.map(String) : [];
    else v[p.key] = p.default === undefined || p.default === null ? "" : String(p.default);
  }
  return v;
}

/** The attributes a field parameter may use. */
function fieldChoices(param: AgentTemplateParam, attributes: Attribute[]): Attribute[] {
  return attributes
    .filter((a) => !param.types?.length || param.types.includes(a.type))
    .filter((a) => !param.localized || a.localized)
    .sort((a, b) => a.name.localeCompare(b.name));
}

/** A number parameter's value: a non-negative integer. */
function isCount(v: Values[string] | undefined): boolean {
  if (typeof v !== "string" || !v.trim()) return false;
  const n = Number(v);
  return Number.isInteger(n) && n >= 0;
}

function emptyValue(v: Values[string] | undefined): boolean {
  return v === undefined || v === "" || (Array.isArray(v) && v.length === 0);
}

/** The body of POST /admin/agent-templates/{id}/render. */
function renderParams(params: AgentTemplateParam[], values: Values, name: string): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const p of params) {
    const v = values[p.key];
    if (p.type === "bool") out[p.key] = v === true;
    else if (emptyValue(v)) continue;
    else if (p.type === "number") out[p.key] = Number(v);
    else out[p.key] = v;
  }
  if (name.trim()) out.name = name.trim();
  return out;
}

function TemplateParamsModal({
  template,
  collections,
  onClose,
  onRendered,
}: {
  template: AgentTemplate;
  collections: CollectionChoice[];
  onClose: () => void;
  onRendered: (definition: AgentDefinition) => void;
}) {
  const [values, setValues] = useState<Values>(() => initialValues(template.params));
  const [name, setName] = useState("");
  const [attempted, setAttempted] = useState(false);
  const [busy, setBusy] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);
  const collectionParam = template.params.find((p) => p.type === "collection");
  const collection = collectionParam ? String(values[collectionParam.key] ?? "") : "";
  const { attributes, loading: attrsLoading, error: attrsError } = useCollectionAttributes(collection || null);
  // Closed (Esc, the overlay, the close button) while rendering: drop the response.
  const open = useRef(true);
  useEffect(() => {
    open.current = true;
    return () => {
      open.current = false;
    };
  }, []);

  const errors: Record<string, string> = {};
  for (const p of template.params) {
    if (p.required && emptyValue(values[p.key])) errors[p.key] = p.type === "collection" ? "Choose a collection." : "Required.";
    if (p.type === "number" && !emptyValue(values[p.key]) && !isCount(values[p.key])) errors[p.key] = "A whole number, 0 or more.";
  }
  if (/[/?#%\\]/.test(name)) errors.name = "The name cannot contain / ? # % or \\ (it is used in URLs).";
  const shown = attempted ? errors : {};

  const setValue = (key: string, value: Values[string]) => setValues((v) => ({ ...v, [key]: value }));
  const setCollection = (key: string, value: string) =>
    // Fields belong to the collection: a new one clears them.
    setValues((v) => {
      const next: Values = { ...v, [key]: value };
      for (const p of template.params) {
        if (p.type === "field") next[p.key] = "";
        if (p.type === "fields") next[p.key] = [];
      }
      return next;
    });

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setAttempted(true);
    setServerError(null);
    if (Object.keys(errors).length > 0) return;
    setBusy(true);
    try {
      const res = await api<AgentDefinition>("POST", `/admin/agent-templates/${encodeURIComponent(template.id)}/render`, {
        params: renderParams(template.params, values, name),
      });
      if (!open.current) return;
      onRendered(res.data);
    } catch (err) {
      if (!open.current) return;
      setServerError(errorMessage(err));
      setBusy(false);
    }
  };

  const renderParam = (p: AgentTemplateParam) => {
    const v = values[p.key];
    const id = `template-param-${p.key}`;
    if (p.type === "bool") {
      return (
        <Field.Root key={p.key} id={id} flexDirection="row" alignItems="start" gap={3}>
          <Switch size="sm" mt={0.5} colorPalette="teal" checked={v === true} onCheckedChange={(e) => setValue(p.key, e.checked)} />
          <Box>
            <Field.Label fontSize="sm">{p.label}</Field.Label>
            {p.help && (
              <Field.HelperText fontSize="sm" mt={0.5}>
                {p.help}
              </Field.HelperText>
            )}
          </Box>
        </Field.Root>
      );
    }
    let control: React.ReactNode;
    if (p.type === "collection") {
      control = (
        <NativeSelect.Root>
          <NativeSelect.Field placeholder="Choose a collection" value={String(v ?? "")} onChange={(e) => setCollection(p.key, e.target.value)}>
            {collections.map((c) => (
              <option key={c.name} value={c.name}>
                {c.name}
              </option>
            ))}
          </NativeSelect.Field>
          <NativeSelect.Indicator />
        </NativeSelect.Root>
      );
    } else if (p.type === "field" || p.type === "fields") {
      const choices = fieldChoices(p, attributes);
      if (!collection) {
        control = (
          <Text fontSize="sm" color="gray.500">
            Choose a collection first.
          </Text>
        );
      } else if (attrsLoading && attributes.length === 0) {
        control = <Spinner size="sm" />;
      } else if (choices.length === 0) {
        control = (
          <Text fontSize="sm" color="orange.600">
            {collection} has no {p.localized ? "localized " : ""}field{p.types?.length ? ` of type ${p.types.join(", ")}` : ""}.
          </Text>
        );
      } else if (p.type === "field") {
        control = (
          <NativeSelect.Root>
            <NativeSelect.Field placeholder={p.required ? "Choose a field" : "None"} value={String(v ?? "")} onChange={(e) => setValue(p.key, e.target.value)}>
              {choices.map((a) => (
                <option key={a.name} value={a.name}>
                  {a.name} ({a.type})
                </option>
              ))}
            </NativeSelect.Field>
            <NativeSelect.Indicator />
          </NativeSelect.Root>
        );
      } else {
        control = (
          <CheckboxGroup value={Array.isArray(v) ? v : []} onValueChange={(next) => setValue(p.key, next.map(String))}>
            <Wrap columnGap={5} rowGap={1}>
              {choices.map((a) => (
                <WrapItem key={a.name}>
                  <Checkbox value={a.name} size="sm">
                    {a.name}
                    <Text as="span" color="gray.500" fontSize="xs" ml={1}>
                      {a.type}
                    </Text>
                  </Checkbox>
                </WrapItem>
              ))}
            </Wrap>
          </CheckboxGroup>
        );
      }
    } else {
      control = (
        <Input
          type={p.type === "number" ? "number" : "text"}
          min={p.type === "number" ? 0 : undefined}
          step={p.type === "number" ? 1 : undefined}
          value={String(v ?? "")}
          onChange={(e) => setValue(p.key, e.target.value)}
        />
      );
    }
    if (p.type === "fields") {
      // Checkboxes must not sit in a Field (they would share its control id).
      return (
        <Fieldset.Root key={p.key} size="sm" invalid={Boolean(shown[p.key])}>
          <Fieldset.Legend fontSize="sm">
            {p.label}
            {p.required && <RequiredMark />}
          </Fieldset.Legend>
          {control}
          {shown[p.key] ? <Fieldset.ErrorText>{shown[p.key]}</Fieldset.ErrorText> : p.help ? <Fieldset.HelperText>{p.help}</Fieldset.HelperText> : null}
        </Fieldset.Root>
      );
    }
    return (
      <Field.Root key={p.key} id={id} required={p.required} invalid={Boolean(shown[p.key])} gap={1}>
        <Field.Label fontSize="sm">
          {p.label}
          <Field.RequiredIndicator />
        </Field.Label>
        {control}
        {shown[p.key] ? (
          <Field.ErrorText fontSize="sm">{shown[p.key]}</Field.ErrorText>
        ) : p.help ? (
          <Field.HelperText fontSize="sm">{p.help}</Field.HelperText>
        ) : null}
      </Field.Root>
    );
  };

  return (
    <DialogRoot
      open
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      size="xl"
      scrollBehavior="inside"
    >
      <DialogContent>
        {/* display: contents keeps header, body and footer laid out by the dialog (scrolling body). */}
        <form onSubmit={submit} noValidate style={{ display: "contents" }}>
          <DialogHeader>
            <DialogTitle>{template.name}</DialogTitle>
          </DialogHeader>
          <DialogBody>
            <VStack align="stretch" gap={4}>
              <Text fontSize="sm" color="gray.600">
                {template.description}
              </Text>
              {template.params.map(renderParam)}
              <Field.Root invalid={Boolean(shown.name)} id="template-agent-name" gap={1}>
                <Field.Label fontSize="sm">Agent name</Field.Label>
                <Input
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder={`${template.id}-${collection || "<collection>"}`}
                  autoComplete="off"
                />
                {shown.name ? (
                  <Field.ErrorText fontSize="sm">{shown.name}</Field.ErrorText>
                ) : (
                  <Field.HelperText fontSize="sm">
                    Empty: {template.id}-{collection || "<collection>"}.
                  </Field.HelperText>
                )}
              </Field.Root>
              {attrsError && (
                <Notice status="error" borderRadius="md">
                  {attrsError}
                </Notice>
              )}
              {serverError && (
                <Notice status="error" borderRadius="md">
                  {serverError}
                </Notice>
              )}
            </VStack>
          </DialogBody>
          <DialogFooter>
            <Button variant="ghost" onClick={onClose} disabled={busy}>
              Cancel
            </Button>
            <Button type="submit" colorPalette="teal" loading={busy} loadingText="Preparing">
              Continue
            </Button>
          </DialogFooter>
        </form>
        <DialogCloseTrigger />
      </DialogContent>
    </DialogRoot>
  );
}

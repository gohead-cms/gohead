import { type FormEvent, useEffect, useRef, useState } from "react";
import {
  Alert,
  AlertIcon,
  Box,
  Button,
  Checkbox,
  CheckboxGroup,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Heading,
  Icon,
  Input,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Select,
  SimpleGrid,
  Spinner,
  Switch,
  Text,
  VStack,
  Wrap,
  WrapItem,
} from "@chakra-ui/react";
import { type IconType } from "react-icons";
import { FiCpu, FiFileText, FiGlobe, FiPlus, FiShield, FiTag } from "react-icons/fi";
import { api } from "../../../services/api";
import { type AgentDefinition, type AgentTemplate, type AgentTemplateParam, type Attribute } from "../../../shared/types";
import { errorMessage } from "../../../shared/utils/errors";
import { type CollectionChoice, useAgentTemplates, useCollectionAttributes } from "../hook/useEditorData";

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
        <Heading size="sm" mb={1}>
          {title}
        </Heading>
        <Text fontSize="sm" color="gray.600">
          {description}
        </Text>
      </Box>
      <Button size="sm" colorScheme="teal" variant="outline" alignSelf="start" onClick={onClick}>
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
    <VStack align="stretch" spacing={4}>
      <Box>
        <Heading size="md">Start from a template</Heading>
        <Text fontSize="sm" color="gray.500" mt={1}>
          A template writes the system prompt, trigger and functions for a common task. You review everything before the agent is created.
        </Text>
      </Box>
      {error && (
        <Alert status="warning" borderRadius="md">
          <AlertIcon />
          {error}
        </Alert>
      )}
      <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} spacing={4}>
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
        <FormControl key={p.key} display="flex" alignItems="start" gap={3}>
          <Switch id={id} mt={0.5} colorScheme="teal" isChecked={v === true} onChange={(e) => setValue(p.key, e.target.checked)} />
          <Box>
            <FormLabel htmlFor={id} mb={0} fontSize="sm">
              {p.label}
            </FormLabel>
            {p.help && <FormHelperText mt={0.5}>{p.help}</FormHelperText>}
          </Box>
        </FormControl>
      );
    }
    let control: React.ReactNode;
    if (p.type === "collection") {
      control = (
        <Select id={id} placeholder="Choose a collection" value={String(v ?? "")} onChange={(e) => setCollection(p.key, e.target.value)}>
          {collections.map((c) => (
            <option key={c.name} value={c.name}>
              {c.name}
            </option>
          ))}
        </Select>
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
          <Select id={id} placeholder={p.required ? "Choose a field" : "None"} value={String(v ?? "")} onChange={(e) => setValue(p.key, e.target.value)}>
            {choices.map((a) => (
              <option key={a.name} value={a.name}>
                {a.name} ({a.type})
              </option>
            ))}
          </Select>
        );
      } else {
        control = (
          <CheckboxGroup value={Array.isArray(v) ? v : []} onChange={(next) => setValue(p.key, next.map(String))}>
            <Wrap spacingX={5} spacingY={1}>
              {choices.map((a) => (
                <WrapItem key={a.name}>
                  <Checkbox value={a.name}>
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
          id={id}
          type={p.type === "number" ? "number" : "text"}
          min={p.type === "number" ? 0 : undefined}
          step={p.type === "number" ? 1 : undefined}
          value={String(v ?? "")}
          onChange={(e) => setValue(p.key, e.target.value)}
        />
      );
    }
    const isGroup = p.type === "fields";
    return (
      <FormControl key={p.key} as={isGroup ? "fieldset" : undefined} isRequired={p.required} isInvalid={Boolean(shown[p.key])}>
        <FormLabel as={isGroup ? "legend" : undefined} htmlFor={isGroup ? undefined : id} fontSize="sm" mb={1}>
          {p.label}
        </FormLabel>
        {control}
        {shown[p.key] ? <FormErrorMessage>{shown[p.key]}</FormErrorMessage> : p.help ? <FormHelperText>{p.help}</FormHelperText> : null}
      </FormControl>
    );
  };

  return (
    <Modal isOpen onClose={onClose} size="xl" scrollBehavior="inside">
      <ModalOverlay />
      <ModalContent as="form" onSubmit={submit} noValidate>
        <ModalHeader>{template.name}</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <VStack align="stretch" spacing={4}>
            <Text fontSize="sm" color="gray.600">
              {template.description}
            </Text>
            {template.params.map(renderParam)}
            <FormControl isInvalid={Boolean(shown.name)}>
              <FormLabel htmlFor="template-agent-name" fontSize="sm" mb={1}>
                Agent name
              </FormLabel>
              <Input
                id="template-agent-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder={`${template.id}-${collection || "<collection>"}`}
                autoComplete="off"
              />
              {shown.name ? <FormErrorMessage>{shown.name}</FormErrorMessage> : <FormHelperText>Empty: {template.id}-{collection || "<collection>"}.</FormHelperText>}
            </FormControl>
            {attrsError && (
              <Alert status="error" borderRadius="md">
                <AlertIcon />
                {attrsError}
              </Alert>
            )}
            {serverError && (
              <Alert status="error" borderRadius="md">
                <AlertIcon />
                {serverError}
              </Alert>
            )}
          </VStack>
        </ModalBody>
        <ModalFooter>
          <Button variant="ghost" mr={3} onClick={onClose} isDisabled={busy}>
            Cancel
          </Button>
          <Button type="submit" colorScheme="teal" isLoading={busy} loadingText="Preparing">
            Continue
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}

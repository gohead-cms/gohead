import { useState } from "react";
import type { FormEvent } from "react";
import { Link as RouterLink, useNavigate } from "react-router-dom";
import {
  Alert,
  AlertDescription,
  AlertIcon,
  Box,
  Button,
  Flex,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Heading,
  Input,
  Switch,
  Text,
  Textarea,
  VStack,
} from "@chakra-ui/react";
import { FiPlus } from "react-icons/fi";
import { api } from "../../../services/api";
import { useNotify } from "../../../shared/hooks/useNotify";
import { PageHeader } from "../../../shared/ui/PageHeader";
import { serverMessage } from "../api";
import type { AttributeForm, ModelKind, TypeForm } from "../schema";
import {
  emptyAttribute,
  formToPayload,
  hasErrors,
  KIND_LABELS,
  moveItem,
  saveRequest,
  typesForKind,
  validateForm,
} from "../schema";
import { AttributeEditor } from "./AttributeEditor";

export interface TypeEditorFormProps {
  kind: ModelKind;
  /** The saved name, or null for a new type. */
  existingName: string | null;
  initial: TypeForm;
  collections: string[];
  components: string[];
  /** Names of other types of this kind. */
  takenNames: string[];
  /** The API did not return every attribute setting (see the warning). */
}

const FORM_ID = "type-editor-form";

/** Name, settings and attributes of a collection, component or singleton. */
export function TypeEditorForm({ kind, existingName, initial, collections, components, takenNames }: TypeEditorFormProps) {
  const navigate = useNavigate();
  const notify = useNotify();
  const [form, setForm] = useState<TypeForm>(initial);
  const [submitted, setSubmitted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const { singular } = KIND_LABELS[kind];
  const canRename = !existingName || kind === "components";
  const errors = validateForm(form, { kind, takenNames });
  const shown = submitted ? errors : { attributes: {} };
  const backTo = kind === "collections" ? "/model" : `/model?tab=${kind}`;

  const set = (patch: Partial<TypeForm>) => setForm((f) => ({ ...f, ...patch }));
  const setAttributes = (update: (attrs: AttributeForm[]) => AttributeForm[]) =>
    setForm((f) => ({ ...f, attributes: update(f.attributes) }));

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    if (hasErrors(errors)) return;
    setSaving(true);
    setServerError(null);
    try {
      const { method, path } = saveRequest(kind, existingName);
      await api(method, path, formToPayload(kind, form));
      notify.success(`Saved ${singular} "${form.name.trim()}"`);
      navigate(backTo);
    } catch (err) {
      setServerError(serverMessage(err));
      setSaving(false);
    }
  };

  const title = existingName ? `Edit ${singular} "${existingName}"` : `New ${singular}`;

  return (
    <form id={FORM_ID} onSubmit={submit} noValidate>
      <PageHeader
        title={title}
        description={existingName ? "Changes replace the saved definition: removed attributes are deleted." : undefined}
        actions={
          <>
            <Button as={RouterLink} to={backTo} variant="ghost" size="sm">
              Cancel
            </Button>
            <Button type="submit" colorScheme="teal" size="sm" isLoading={saving} loadingText="Saving">
              Save
            </Button>
          </>
        }
      />
      <VStack align="stretch" spacing={6}>
        {serverError && (
          <Alert status="error" borderRadius="md">
            <AlertIcon />
            <AlertDescription>{serverError}</AlertDescription>
          </Alert>
        )}
        {submitted && hasErrors(errors) && (
          <Alert status="warning" borderRadius="md">
            <AlertIcon />
            <AlertDescription>{errors.general ?? "Fix the highlighted fields before saving."}</AlertDescription>
          </Alert>
        )}

        <Box bg="white" borderWidth="1px" borderRadius="lg" p={6}>
          <Heading size="sm" mb={4}>
            Settings
          </Heading>
          <VStack align="stretch" spacing={4}>
            <FormControl isRequired isInvalid={!!shown.name}>
              <FormLabel>Name</FormLabel>
              <Input
                value={form.name}
                onChange={(e) => set({ name: e.target.value })}
                isReadOnly={!canRename}
                bg={canRename ? undefined : "gray.100"}
                placeholder={kind === "collections" ? "articles" : kind === "components" ? "seo" : "homepage"}
                maxW="md"
              />
              {shown.name ? (
                <FormErrorMessage>{shown.name}</FormErrorMessage>
              ) : (
                <FormHelperText>{nameHelp(kind, !!existingName)}</FormHelperText>
              )}
            </FormControl>
            <FormControl>
              <FormLabel>Description</FormLabel>
              <Textarea rows={2} value={form.description} onChange={(e) => set({ description: e.target.value })} />
            </FormControl>
            {kind === "collections" && (
              <FormControl display="flex" flexDirection="column">
                <Flex align="center" gap={3}>
                  <Switch
                    id="draft-and-publish"
                    isChecked={form.draftAndPublish}
                    onChange={(e) => set({ draftAndPublish: e.target.checked })}
                    colorScheme="teal"
                  />
                  <FormLabel htmlFor="draft-and-publish" mb={0}>
                    Draft and publish
                  </FormLabel>
                </Flex>
                <FormHelperText>
                  New items start as drafts and readers only see published items. Turning it on publishes existing items;
                  turning it off makes drafts visible.
                </FormHelperText>
              </FormControl>
            )}
          </VStack>
        </Box>

        <Box bg="white" borderWidth="1px" borderRadius="lg" p={6}>
          <Flex justify="space-between" align="center" mb={1} gap={4} wrap="wrap">
            <Heading size="sm">Attributes</Heading>
            <Button size="sm" leftIcon={<FiPlus />} onClick={() => setAttributes((a) => [...a, emptyAttribute()])}>
              Add attribute
            </Button>
          </Flex>
          <Text fontSize="sm" color="gray.500" mb={4}>
            The API stores attributes by name, so the order here is not saved.
          </Text>
          <VStack align="stretch" spacing={3}>
            {form.attributes.length === 0 && (
              <Text color="gray.500" textAlign="center" py={6}>
                No attributes yet. Add at least one.
              </Text>
            )}
            {form.attributes.map((attr, i) => (
              <AttributeEditor
                key={attr.key}
                attr={attr}
                index={i}
                count={form.attributes.length}
                types={typesForKind(kind)}
                errors={shown.attributes[attr.key] ?? {}}
                collections={collections}
                components={kind === "components" ? components.filter((c) => c !== existingName) : components}
                onChange={(patch) => setAttributes((list) => list.map((a) => (a.key === attr.key ? { ...a, ...patch } : a)))}
                onMove={(delta) => setAttributes((list) => moveItem(list, list.findIndex((a) => a.key === attr.key), delta))}
                onRemove={() => setAttributes((list) => list.filter((a) => a.key !== attr.key))}
              />
            ))}
          </VStack>
        </Box>

        <Flex justify="flex-end" gap={2}>
          <Button as={RouterLink} to={backTo} variant="ghost">
            Cancel
          </Button>
          <Button type="submit" colorScheme="teal" isLoading={saving} loadingText="Saving">
            Save {singular}
          </Button>
        </Flex>
      </VStack>
    </form>
  );
}

function nameHelp(kind: ModelKind, existing: boolean): string {
  if (existing && kind === "collections") return "A collection cannot be renamed: relations and URLs refer to it by name.";
  if (existing && kind === "singletons") return "A singleton cannot be renamed.";
  if (kind === "components") return existing ? "Renaming is refused while the component is in use." : "Referenced by component attributes, e.g. seo.";
  if (kind === "collections") return "Used in URLs (/api/collections/{name}) and GraphQL. Lower-case plural, e.g. articles.";
  return "Used in URLs (/api/singleton/{name}) and GraphQL.";
}

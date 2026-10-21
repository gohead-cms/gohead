import { useMemo, useState } from "react";
import { Alert, Badge, Box, Button, Flex, Heading, HStack, Icon, Text, chakra } from "@chakra-ui/react";
import { FiClock, FiSave } from "react-icons/fi";
import { api } from "../../../services/api";
import { localeLabel, useLocales } from "../../../shared/hooks/useLocales";
import { useNotify } from "../../../shared/hooks/useNotify";
import { PageLoader } from "../../../shared/ui/page-loader";
import type { Attribute } from "../../../shared/types";
import { formatTime } from "../../agents/components/runFormat";
import { useMe } from "../../auth/me";
import { singletonResource } from "../../auth/permissions";
import { HistoryDrawer } from "../../history/HistoryDrawer";
import { type SingletonContent, type SingletonDef, singletonPath, useSingletonContent } from "../hooks/useSingletons";
import { EntryFields } from "./EntryFields";
import { attributeMap, entryLocales, hasLocalizedAttributes, initialValues, missingRequired, toEntryPayload } from "./entryValues";
import { LocaleSwitcher } from "./LocaleSwitcher";
import { TranslationForm } from "./TranslationForm";
import { useLocaleGuard } from "./useLocaleGuard";

/** Loads a singleton's content and shows its form. */
export function SingletonEditor({ singleton }: { singleton: SingletonDef }) {
  const { content, loading, error, refetch } = useSingletonContent(singleton.name);
  const { can } = useMe();
  const canUpdate = can("update", singletonResource(singleton.name));
  const [historyOpen, setHistoryOpen] = useState(false);
  const attributes = useMemo(() => attributeMap(singleton.attributes ?? undefined), [singleton.attributes]);
  // While saved content reloads, the form stays on screen.
  const ready = !loading || !!content;
  const localized = hasLocalizedAttributes(attributes);
  const { locales, defaultLocale } = useLocales(localized);
  const [picked, setPicked] = useState<string | null>(null);
  // Switching locale replaces the form: unsaved edits are confirmed first.
  const guard = useLocaleGuard(setPicked);
  const reload = () => {
    guard.markClean();
    refetch();
  };
  const translatable = localized && locales.length > 1 && !!defaultLocale;
  // Translations need content in the default locale first.
  const active = translatable && content && picked ? (locales.find((l) => l.code === picked && !l.isDefault) ?? null) : null;

  return (
    <Box>
      <Flex justify="space-between" align="center" mb={6} gap={4} wrap="wrap">
        <Box>
          <HStack>
            <Heading size="2xl" fontWeight="bold">{singleton.name}</Heading>
            <Badge colorPalette="purple">Single type</Badge>
          </HStack>
          <Text color="gray.500">
            {singleton.description ||
              (content?.updatedAt ? `Last updated ${formatTime(content.updatedAt)}` : "One entry, edited in place.")}
          </Text>
        </Box>
        <HStack>
          {translatable && defaultLocale && (
            <LocaleSwitcher
              locales={locales}
              value={active?.code ?? defaultLocale.code}
              onChange={guard.request}
              translated={content ? entryLocales(content) : undefined}
              disabledReason={content ? undefined : `Save the content in ${localeLabel(defaultLocale)} first, then translate it.`}
            />
          )}
          {/* Also without content: deleted content can be restored from history. */}
          {/* History and restores need update on the single type. */}
          {canUpdate && (
            <Button bg="white" variant="outline" size="xs" onClick={() => setHistoryOpen(true)}>
              <Icon as={FiClock} />
              History
            </Button>
          )}
        </HStack>
      </Flex>
      {!ready && <PageLoader text="Fetching content..." />}
      {error && (
        <Alert.Root status="error" borderRadius="md">
          <Alert.Indicator />
          <Alert.Description>{error}</Alert.Description>
        </Alert.Root>
      )}
      {ready && !error && active && defaultLocale && (
        <Box bg="white" borderWidth="1px" borderRadius="lg" p={6}>
          <TranslationForm
            key={active.code}
            path={singletonPath(singleton.name)}
            readQuery="status=all"
            attributes={attributes}
            locale={active}
            defaultLocale={defaultLocale}
            canWrite={canUpdate}
            idPrefix={`single-${singleton.name}-`}
            onEdit={guard.markDirty}
            onSaved={reload}
          />
        </Box>
      )}
      {ready && !error && !active && (
        <SingletonForm
          // A new version of the content starts the form again.
          key={`${singleton.name}-${content?.updatedAt ?? "new"}`}
          name={singleton.name}
          attributes={attributes}
          content={content ?? null}
          canUpdate={canUpdate}
          onEdit={guard.markDirty}
          onSaved={reload}
        />
      )}
      <HistoryDrawer
        target={historyOpen ? { kind: "singleton", name: singleton.name } : null}
        onClose={() => setHistoryOpen(false)}
        // The form restarts from the reloaded content (its key is the update time).
        onRestored={reload}
      />
      {guard.dialog}
    </Box>
  );
}

interface SingletonFormProps {
  name: string;
  attributes: Record<string, Attribute>;
  content: SingletonContent | null;
  /** False shows the content read-only. */
  canUpdate: boolean;
  onSaved: () => void;
  /** Called whenever a field is edited. */
  onEdit: () => void;
}

function SingletonForm({ name, attributes, content, canUpdate, onSaved, onEdit }: SingletonFormProps) {
  const notify = useNotify();
  const [values, setValues] = useState<Record<string, unknown>>(() => initialValues(attributes, content));
  const [saving, setSaving] = useState(false);
  const [missing, setMissing] = useState<string[]>([]);

  if (Object.keys(attributes).length === 0) {
    return <Text color="gray.500">This single type has no attributes yet. Add some in the content model.</Text>;
  }

  const onChange = (key: string, value: unknown) => {
    onEdit();
    setValues((prev) => ({ ...prev, [key]: value }));
  };

  const save = async () => {
    const empty = missingRequired(attributes, values);
    setMissing(empty);
    if (empty.length > 0) return;
    setSaving(true);
    try {
      await api("PUT", singletonPath(name), { data: toEntryPayload(attributes, values) });
      notify.success(content ? "Changes saved" : "Content created");
      onSaved();
    } catch (err) {
      notify.failure(err, "Could not save");
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box bg="white" borderWidth="1px" borderRadius="lg" p={6}>
      {!content && (
        <Alert.Root status="info" borderRadius="md" mb={6}>
          <Alert.Indicator />
          <Alert.Description>{canUpdate ? "No content yet: fill in the fields and save to create it." : "No content yet."}</Alert.Description>
        </Alert.Root>
      )}
      {missing.length > 0 && (
        <Alert.Root status="warning" borderRadius="md" mb={6}>
          <Alert.Indicator />
          <Alert.Description>Required: {missing.join(", ")}</Alert.Description>
        </Alert.Root>
      )}
      <chakra.fieldset disabled={!canUpdate || saving} minW={0}>
        <EntryFields attributes={attributes} values={values} onChange={onChange} idPrefix={`single-${name}-`} />
      </chakra.fieldset>
      <Flex justify="flex-end" mt={6} gap={3} align="center">
        {!canUpdate && <Text fontSize="sm" color="gray.500">Your role can read this single type but not change it.</Text>}
        {canUpdate && (
          <Button colorPalette="purple" onClick={save} loading={saving}>
            <Icon as={FiSave} />
            Save
          </Button>
        )}
      </Flex>
    </Box>
  );
}

import { type ReactNode, useMemo, useState } from "react";
import { Alert, AlertDescription, AlertIcon, Box, Button, Code, HStack, Icon, Spacer, Spinner, Text, VStack } from "@chakra-ui/react";
import { FiSave, FiTrash2 } from "react-icons/fi";
import { api } from "../../../services/api";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import { type Locale, localeLabel } from "../../../shared/hooks/useLocales";
import { useNotify } from "../../../shared/hooks/useNotify";
import type { Attribute } from "../../../shared/types";
import { ConfirmButton } from "../../../shared/ui/ConfirmButton";
import { EntryFields } from "./EntryFields";
import { entryLocales, hintValue, localizedAttributes, toTranslationPayload, translationValues } from "./entryValues";

type Entry = Record<string, unknown>;

/** The form's buttons: start (remove the translation) and end (save). */
export interface TranslationActions {
  start: ReactNode;
  end: ReactNode;
}

/** Places the form's fields and its buttons (e.g. a drawer's body and footer). */
export type TranslationLayout = (body: ReactNode, actions: TranslationActions) => ReactNode;

interface TranslationFormProps {
  /** The entry's API path, without a query (an item or a singleton). */
  path: string;
  /** Extra query parameters of reads, e.g. "status=all" for singletons. */
  readQuery?: string;
  attributes: Record<string, Attribute>;
  locale: Locale;
  defaultLocale: Locale;
  canWrite: boolean;
  /** Called after the translation is saved or removed. */
  onSaved?: () => void;
  /** Called whenever a field is edited (the form has unsaved changes). */
  onEdit?: () => void;
  /** Extra buttons before the form's own (e.g. Cancel). */
  extraActions?: ReactNode;
  layout?: TranslationLayout;
  idPrefix?: string;
}

const stackLayout: TranslationLayout = (body, actions) => (
  <>
    {body}
    <HStack mt={6} spacing={3}>
      {actions.start}
      <Spacer />
      {actions.end}
    </HStack>
  </>
);

/**
 * Edits an entry's translation into one locale: only its localized
 * attributes, with the default locale's value under each field. Loads the
 * entry in both locales (the translation without fallback).
 */
export function TranslationForm(props: TranslationFormProps) {
  const { path, readQuery, locale, defaultLocale } = props;
  const q = (extra: string) => `${path}?level=0${readQuery ? `&${readQuery}` : ""}${extra}`;
  const translationPath = q(`&locale=${encodeURIComponent(locale.code)}&fallback=false`);
  const { data, loading, error, refetch } = useApiQuery(translationPath, async () => {
    const [def, tr] = await Promise.all([
      getJSON<Entry>(q(""), `Failed to load the ${defaultLocale.code} content`),
      getJSON<Entry>(translationPath, `Failed to load the ${locale.code} translation`),
    ]);
    return { defaults: def.data ?? {}, translation: tr.data ?? {} };
  });
  const layout = props.layout ?? stackLayout;

  if (error) {
    return layout(
      <Alert status="error" borderRadius="md">
        <AlertIcon />
        <AlertDescription flex="1">{error}</AlertDescription>
        <Button size="sm" onClick={refetch}>
          Retry
        </Button>
      </Alert>,
      { start: null, end: props.extraActions },
    );
  }
  if (!data) {
    return layout(
      <HStack py={10} justify="center" color="gray.500">
        <Spinner size="sm" />
        <Text>Loading the {locale.code} translation…</Text>
      </HStack>,
      { start: null, end: props.extraActions },
    );
  }
  return (
    <TranslationFields
      // A reloaded translation (saved or removed) starts the form again.
      key={JSON.stringify(entryLocales(data.translation)) + JSON.stringify(translationValues(localizedAttributes(props.attributes), data.translation))}
      {...props}
      layout={layout}
      defaults={data.defaults}
      translation={data.translation}
      loading={loading}
      reload={refetch}
    />
  );
}

interface FieldsProps extends TranslationFormProps {
  layout: TranslationLayout;
  defaults: Entry;
  translation: Entry;
  loading: boolean;
  reload: () => void;
}

function TranslationFields({
  path,
  attributes,
  locale,
  defaultLocale,
  canWrite,
  onSaved,
  onEdit,
  extraActions,
  layout,
  idPrefix = "tr-",
  defaults,
  translation,
  reload,
}: FieldsProps) {
  const notify = useNotify();
  const localized = useMemo(() => localizedAttributes(attributes), [attributes]);
  // Not required here: an empty field falls back to the default locale.
  const fields = useMemo(
    () => Object.fromEntries(Object.entries(localized).map(([k, a]) => [k, { ...a, required: false }])),
    [localized],
  );
  const shared = Object.keys(attributes).filter((k) => !(k in localized));
  const [values, setValues] = useState<Entry>(() => translationValues(localized, translation));
  const [saving, setSaving] = useState(false);
  const translated = entryLocales(translation).includes(locale.code);
  const target = `${path}?locale=${encodeURIComponent(locale.code)}`;
  const def = defaultLocale.code;

  const setValue = (key: string, value: unknown) => {
    onEdit?.();
    setValues((prev) => ({ ...prev, [key]: value }));
  };
  // Clearing a field drops its translated value: it falls back to the
  // default locale (the only way back for a switch or a choice).
  const clearValue = (key: string) => {
    onEdit?.();
    setValues((prev) => {
      const next = { ...prev };
      delete next[key];
      return next;
    });
  };

  const hints = Object.fromEntries(
    Object.keys(localized).map((k) => [
      k,
      hasValue(values[k]) ? (
        <HStack as="span" spacing={2} align="baseline">
          <Text as="span" noOfLines={2}>
            Default ({def}): <Text as="span" color="gray.600">{hintValue(defaults[k])}</Text>
          </Text>
          <Button size="xs" variant="link" colorScheme="purple" flexShrink={0} title={`Clear the ${locale.code} ${k}: it falls back to the ${def} value`} onClick={() => clearValue(k)}>
            Use default
          </Button>
        </HStack>
      ) : (
        <Text as="span" noOfLines={2}>
          Falls back to the default ({def}): <Text as="span" color="gray.600">{hintValue(defaults[k])}</Text>
        </Text>
      ),
    ]),
  );
  const placeholders = useMemo(
    () =>
      Object.fromEntries(
        Object.keys(localized)
          .filter((k) => typeof defaults[k] === "string" && defaults[k])
          .map((k) => [k, hintValue(defaults[k])]),
      ),
    [localized, defaults],
  );

  const save = async () => {
    setSaving(true);
    try {
      await api("PUT", target, { data: toTranslationPayload(localized, values) });
      notify.success(`${locale.code} translation saved`);
      reload();
      onSaved?.();
    } catch (err) {
      notify.failure(err, `Could not save the ${locale.code} translation`);
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    try {
      await api("PUT", target, { data: {} });
      notify.success(`${locale.code} translation removed`);
      reload();
      onSaved?.();
    } catch (err) {
      notify.failure(err, `Could not remove the ${locale.code} translation`);
    }
  };

  const body = (
    <VStack align="stretch" spacing={6}>
      <Alert status="info" borderRadius="md" alignItems="start">
        <AlertIcon />
        <AlertDescription fontSize="sm">
          <Text>
            Translating into <strong>{localeLabel(locale)}</strong>
            {translated ? "." : ": no translation yet."} A field left empty falls back to the {localeLabel(defaultLocale)} value.
          </Text>
          {shared.length > 0 && (
            <Text mt={1}>
              Shared attributes ({shared.map((s, i) => (
                <span key={s}>
                  {i > 0 && ", "}
                  <Code fontSize="xs">{s}</Code>
                </span>
              ))}) are edited in {def}.
            </Text>
          )}
        </AlertDescription>
      </Alert>
      <Box as="fieldset" disabled={!canWrite || saving} minW={0}>
        <EntryFields
          attributes={fields}
          values={values}
          onChange={setValue}
          idPrefix={`${idPrefix}${locale.code}-`}
          hints={hints}
          placeholders={placeholders}
        />
      </Box>
    </VStack>
  );

  const start = canWrite && translated && (
        <ConfirmButton
          title={`Remove the ${locale.code} translation?`}
          body={`Every localized value in ${localeLabel(locale)} is deleted: readers of ${locale.code} get the ${def} content instead. Shared attributes are not affected.`}
          confirmLabel="Remove translation"
          onConfirm={remove}
          trigger={
            <Button variant="ghost" colorScheme="red" leftIcon={<Icon as={FiTrash2} />} isDisabled={saving}>
              Remove {locale.code} translation
            </Button>
          }
        />
  );
  const end = (
    <>
      {extraActions}
      {canWrite && (
        <Button colorScheme="purple" leftIcon={<Icon as={FiSave} />} onClick={save} isLoading={saving}>
          Save translation
        </Button>
      )}
    </>
  );

  return layout(body, { start, end });
}

/** Whether a field holds a translated value (an empty one falls back). */
function hasValue(v: unknown): boolean {
  return v !== undefined && v !== null && v !== "" && !(Array.isArray(v) && v.length === 0);
}

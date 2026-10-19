import { type ReactNode, useState } from "react";
import {
  Alert,
  AlertIcon,
  Drawer,
  DrawerOverlay,
  DrawerContent,
  DrawerHeader,
  DrawerCloseButton,
  DrawerBody,
  DrawerFooter,
  Button,
  Flex,
  Spacer,
  Text,
} from "@chakra-ui/react";
import { FiClock } from "react-icons/fi";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import { type Locale, localeLabel } from "../../../shared/hooks/useLocales";
import type { Schema } from "../../../shared/types";
import { PageLoader } from "../../../shared/ui/page-loader";
import { useMe } from "../../auth/me";
import type { ContentItem } from "../hooks/useCollectionContent";
import { EntryFields } from "./EntryFields";
import { entryLocales, hasLocalizedAttributes, initialValues, toEntryPayload } from "./entryValues";
import { LocaleSwitcher } from "./LocaleSwitcher";
import { TranslationForm } from "./TranslationForm";

interface EntryFormDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  schema: Schema;
  onSubmit: (payload: { data: Partial<ContentItem> }, id?: string | number) => Promise<void>;
  itemToEdit?: ContentItem | null; // Optional prop for the item being edited
  /** Opens the history of the entry being edited (editors and admins). */
  onShowHistory?: (item: ContentItem) => void;
  /** The collection, for the entry's API path. */
  collection: string;
  /** The instance's locales; with more than one, localized types get a locale switcher. */
  locales?: Locale[];
  /** The locale the form opens in (default: the default locale). */
  initialLocale?: string | null;
  /** Called after a translation is saved or removed. */
  onTranslationSaved?: () => void;
}

/**
 * Creates or edits an entry. The form starts from schema and itemToEdit when
 * it mounts: give it a new key each time it is opened for another entry.
 * Entries are created in the default locale; an existing entry of a type
 * with localized attributes can be switched to another locale to edit its
 * translation.
 */
export function EntryFormModal({
  isOpen,
  onClose,
  schema,
  itemToEdit,
  onSubmit,
  onShowHistory,
  collection,
  locales = [],
  initialLocale,
  onTranslationSaved,
}: EntryFormDrawerProps) {
  const { canWrite } = useMe();
  const [picked, setPicked] = useState<string | null>(initialLocale ?? null);
  const defaultLocale = locales.find((l) => l.isDefault) ?? null;
  const showLocales = locales.length > 1 && !!defaultLocale && hasLocalizedAttributes(schema.attributes);
  const active = showLocales && itemToEdit && picked ? (locales.find((l) => l.code === picked && !l.isDefault) ?? null) : null;
  const path = itemToEdit ? `/api/collections/${encodeURIComponent(collection)}/${encodeURIComponent(String(itemToEdit.id))}` : null;

  const historyButton = itemToEdit && onShowHistory && (
    <Button variant="ghost" leftIcon={<FiClock />} onClick={() => onShowHistory(itemToEdit)}>
      History
    </Button>
  );
  const cancelButton = (
    <Button variant="outline" onClick={onClose}>
      Cancel
    </Button>
  );

  return (
    <Drawer isOpen={isOpen} placement="right" onClose={onClose} size="lg">
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader borderBottomWidth="1px" pr={12}>
          <Flex align="center" gap={3} wrap="wrap">
            <Text as="span">
              {itemToEdit ? "Edit Entry" : "Add New Entry"} in {schema.info?.displayName || schema.collectionName}
            </Text>
            {showLocales && defaultLocale && (
              <LocaleSwitcher
                locales={locales}
                value={active?.code ?? defaultLocale.code}
                onChange={setPicked}
                translated={itemToEdit ? entryLocales(itemToEdit) : undefined}
                disabledReason={
                  itemToEdit ? undefined : `New entries are created in ${localeLabel(defaultLocale)}: translate them once saved.`
                }
              />
            )}
          </Flex>
        </DrawerHeader>
        {active && defaultLocale && path ? (
          <TranslationForm
            key={active.code}
            path={path}
            attributes={schema.attributes}
            locale={active}
            defaultLocale={defaultLocale}
            canWrite={canWrite}
            idPrefix="entry-"
            extraActions={cancelButton}
            onSaved={() => {
              onTranslationSaved?.();
              onClose();
            }}
            layout={(body, actions) => (
              <>
                <DrawerBody>{body}</DrawerBody>
                <DrawerFooter borderTopWidth="1px" gap={3} flexWrap="wrap">
                  {historyButton}
                  {actions.start}
                  <Spacer />
                  {actions.end}
                </DrawerFooter>
              </>
            )}
          />
        ) : (
          <DefaultLocaleEntry
            schema={schema}
            item={itemToEdit ?? null}
            path={path}
            // A list read in another locale holds translated values: the
            // default-locale content is loaded instead.
            reload={!!itemToEdit && !!defaultLocale && typeof itemToEdit.locale === "string" && itemToEdit.locale !== defaultLocale.code}
            onSubmit={onSubmit}
            onClose={onClose}
            start={historyButton}
            cancel={cancelButton}
          />
        )}
      </DrawerContent>
    </Drawer>
  );
}

interface DefaultLocaleEntryProps {
  schema: Schema;
  item: ContentItem | null;
  path: string | null;
  reload: boolean;
  onSubmit: EntryFormDrawerProps["onSubmit"];
  onClose: () => void;
  start: ReactNode;
  cancel: ReactNode;
}

/** The entry's main (default-locale) content, loaded again when the item given is in another locale. */
function DefaultLocaleEntry(props: DefaultLocaleEntryProps) {
  const key = props.reload && props.path ? `${props.path}?level=0` : null;
  const { data, error } = useApiQuery(key, async () => {
    const res = await getJSON<ContentItem>(key!, "Failed to load the entry");
    return res.data;
  });
  if (!key) return <DefaultLocaleForm {...props} entry={props.item} />;
  if (error) {
    return (
      <DrawerBody>
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          {error}
        </Alert>
      </DrawerBody>
    );
  }
  if (!data) {
    return (
      <DrawerBody>
        <PageLoader text="Loading the entry..." />
      </DrawerBody>
    );
  }
  return <DefaultLocaleForm {...props} entry={data} />;
}

function DefaultLocaleForm({ schema, item, entry, onSubmit, onClose, start, cancel }: DefaultLocaleEntryProps & { entry: ContentItem | null }) {
  const [formData, setFormData] = useState<Record<string, unknown>>(() => initialValues(schema.attributes, entry));
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleChange = (key: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
    try {
      await onSubmit({ data: toEntryPayload(schema.attributes, formData) as Partial<ContentItem> }, item?.id);
      onClose();
    } catch {
      // The parent shows the error.
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <>
      <DrawerBody>
        <EntryFields attributes={schema.attributes} values={formData} onChange={handleChange} idPrefix="entry-" />
      </DrawerBody>
      <DrawerFooter borderTopWidth="1px" gap={3} flexWrap="wrap">
        {start}
        <Spacer />
        {cancel}
        <Button colorScheme="purple" onClick={handleSubmit} isLoading={isSubmitting}>
          Save Changes
        </Button>
      </DrawerFooter>
    </>
  );
}

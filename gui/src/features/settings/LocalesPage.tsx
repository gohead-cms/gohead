import { useRef, useState } from "react";
import {
  Alert,
  AlertDescription,
  AlertDialog,
  AlertDialogBody,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogOverlay,
  AlertIcon,
  Badge,
  Box,
  Button,
  Code,
  HStack,
  IconButton,
  Skeleton,
  Stack,
  Table,
  TableContainer,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tooltip,
  Tr,
} from "@chakra-ui/react";
import { FiEdit2, FiGlobe, FiPlus, FiStar, FiTrash2 } from "react-icons/fi";
import { api, ApiError } from "../../services/api";
import { useApiQuery } from "../../shared/hooks/useApiQuery";
import { type Locale, loadLocales, LOCALES_PATH } from "../../shared/hooks/useLocales";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { LocaleFormModal } from "./components/LocaleFormModal";
import { optTime } from "./components/settingsFormat";

/** Editing state: closed, adding (null) or renaming a locale. */
type Editing = { locale: Locale | null } | undefined;

/** A delete refused because entries have content in the locale. */
interface InUse {
  locale: Locale;
  entries: number;
}

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

/** The entries count of a 409 refusing to delete a locale with content. */
function entriesInUse(err: unknown): number | null {
  if (!(err instanceof ApiError) || err.status !== 409) return null;
  const d = err.details as { entries?: unknown } | undefined;
  return typeof d?.entries === "number" ? d.entries : null;
}

export function LocalesPage() {
  const notify = useNotify();
  const [editing, setEditing] = useState<Editing>(undefined);
  const [formKey, setFormKey] = useState(0);
  const [inUse, setInUse] = useState<InUse | null>(null);
  const { data, loading, error, refetch } = useApiQuery(LOCALES_PATH, loadLocales);
  const locales = data ?? [];
  const defaultLocale = locales.find((l) => l.isDefault);

  const openForm = (locale: Locale | null) => {
    setFormKey((k) => k + 1);
    setEditing({ locale });
  };

  const makeDefault = async (l: Locale) => {
    try {
      await api("PUT", `/admin/locales/${encodeURIComponent(l.code)}`, { isDefault: true });
      notify.success(`${l.code} is now the default locale`);
      refetch();
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        notify.warning(
          `Could not make ${l.code} the default`,
          `${err.message}. The main content can only be relabelled while no entry is translated into ${l.code}: remove those translations first.`,
        );
        return;
      }
      notify.failure(err, `Could not make ${l.code} the default`);
    }
  };

  const remove = async (l: Locale, purge = false) => {
    try {
      const res = await api("DELETE", `/admin/locales/${encodeURIComponent(l.code)}${purge ? "?purge=true" : ""}`);
      const purged = Number(res.meta?.purgedEntries ?? 0);
      notify.success(
        `Locale ${l.code} deleted`,
        purged > 0 ? `${plural(purged, "translation was", "translations were")} deleted with it.` : undefined,
      );
      refetch();
    } catch (err) {
      const entries = entriesInUse(err);
      if (!purge && entries !== null) {
        setInUse({ locale: l, entries });
        return;
      }
      notify.failure(err, `Could not delete ${l.code}`);
    }
  };

  const addButton = (
    <Button leftIcon={<FiPlus />} colorScheme="blue" onClick={() => openForm(null)}>
      Add locale
    </Button>
  );

  return (
    <Page>
      <PageHeader
        title="Locales"
        description="Languages content can be translated into. Attributes marked as localized in the content model get a value per locale; the others are shared. Entries are written in the default locale, then translated."
        actions={addButton}
      />

      {error && (
        <Alert status="error" borderRadius="md" mb={4}>
          <AlertIcon />
          <AlertDescription flex="1">{error}</AlertDescription>
          <Button size="sm" onClick={refetch}>
            Retry
          </Button>
        </Alert>
      )}

      {loading && !data ? (
        <Stack bg="white" borderWidth="1px" borderRadius="lg" p={4}>
          {[0, 1].map((i) => (
            <Skeleton key={i} h="32px" />
          ))}
        </Stack>
      ) : data && locales.length === 0 ? (
        <EmptyState icon={FiGlobe} title="No locales" action={addButton}>
          Add the language your content is written in first: it becomes the default locale.
        </EmptyState>
      ) : data ? (
        <>
          <Box bg="white" borderWidth="1px" borderRadius="lg" overflow="hidden">
            <TableContainer whiteSpace="normal">
              <Table size="sm">
                <Thead>
                  <Tr>
                    <Th>Code</Th>
                    <Th>Name</Th>
                    <Th>Added</Th>
                    <Th textAlign="right">Actions</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {locales.map((l) => (
                    <Tr key={l.code}>
                      <Td>
                        <HStack>
                          <Code fontSize="sm">{l.code}</Code>
                          {l.isDefault && <Badge colorScheme="purple">Default</Badge>}
                        </HStack>
                      </Td>
                      <Td fontWeight="medium">{l.name || <Text as="span" color="gray.400">—</Text>}</Td>
                      <Td color="gray.600">{optTime(l.createdAt)}</Td>
                      <Td>
                        <HStack justify="flex-end" spacing={1}>
                          {!l.isDefault && (
                            <ConfirmButton
                              title={`Make ${l.code} the default locale?`}
                              confirmLabel="Make default"
                              confirmColorScheme="purple"
                              onConfirm={() => makeDefault(l)}
                              body={
                                <Stack spacing={3}>
                                  <Text>
                                    Entries' main content is relabelled as <strong>{l.code}</strong>: nothing is translated or moved,
                                    so do this when the content is actually written in {l.name || l.code}
                                    {defaultLocale ? ` rather than ${defaultLocale.name || defaultLocale.code}` : ""}.
                                  </Text>
                                  <Text color="gray.600" fontSize="sm">
                                    It is only possible while no entry has a translation in {l.code}. New entries will be created in{" "}
                                    {l.code}.
                                  </Text>
                                </Stack>
                              }
                              trigger={
                                <Button size="xs" variant="ghost" leftIcon={<FiStar />} aria-label={`Make ${l.code} the default`}>
                                  Make default
                                </Button>
                              }
                            />
                          )}
                          <Tooltip label="Rename">
                            <IconButton aria-label={`Rename ${l.code}`} icon={<FiEdit2 />} size="sm" variant="ghost" onClick={() => openForm(l)} />
                          </Tooltip>
                          {l.isDefault ? (
                            <Tooltip label="The default locale cannot be deleted: make another locale the default first." hasArrow>
                              <span>
                                <IconButton aria-label={`Delete ${l.code}`} icon={<FiTrash2 />} size="sm" variant="ghost" colorScheme="red" isDisabled />
                              </span>
                            </Tooltip>
                          ) : (
                            <ConfirmButton
                              title={`Delete locale ${l.code}?`}
                              body={`Clients can no longer read or write content in ${l.name || l.code}. If entries have translations in it, you will be asked before they are deleted.`}
                              onConfirm={() => remove(l)}
                              trigger={<IconButton aria-label={`Delete ${l.code}`} icon={<FiTrash2 />} size="sm" variant="ghost" colorScheme="red" />}
                            />
                          )}
                        </HStack>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </TableContainer>
          </Box>
          {locales.length === 1 && (
            <Text fontSize="sm" color="gray.500" mt={3}>
              Add a second locale to start translating: the content editor then shows a locale switcher for types with localized
              attributes.
            </Text>
          )}
        </>
      ) : null}

      <LocaleFormModal
        key={formKey}
        isOpen={editing !== undefined}
        locale={editing?.locale ?? null}
        existingCodes={locales.map((l) => l.code)}
        onClose={() => setEditing(undefined)}
        onSaved={(l) => {
          notify.success(editing?.locale ? `Locale ${l.code} renamed` : `Locale ${l.code} added`);
          setEditing(undefined);
          refetch();
        }}
      />

      <PurgeDialog
        inUse={inUse}
        onClose={() => setInUse(null)}
        onConfirm={async () => {
          if (inUse) await remove(inUse.locale, true);
          setInUse(null);
        }}
      />
    </Page>
  );
}

/** Offers to delete a locale together with the translations it still has. */
function PurgeDialog({ inUse, onClose, onConfirm }: { inUse: InUse | null; onClose: () => void; onConfirm: () => Promise<void> }) {
  const cancelRef = useRef<HTMLButtonElement>(null);
  const [busy, setBusy] = useState(false);
  const confirm = async () => {
    setBusy(true);
    try {
      await onConfirm();
    } finally {
      setBusy(false);
    }
  };
  const n = inUse?.entries ?? 0;
  return (
    <AlertDialog isOpen={!!inUse} leastDestructiveRef={cancelRef} onClose={onClose}>
      <AlertDialogOverlay>
        <AlertDialogContent>
          <AlertDialogHeader fontSize="lg" fontWeight="bold">
            {inUse?.locale.code} has translations
          </AlertDialogHeader>
          <AlertDialogBody>
            <Text>
              {plural(n, "entry has", "entries have")} content in {inUse?.locale.name || inUse?.locale.code}. Deleting the locale
              deletes {n === 1 ? "this translation" : "these translations"} too; the default-locale content is kept.
            </Text>
          </AlertDialogBody>
          <AlertDialogFooter>
            <Button ref={cancelRef} onClick={onClose} isDisabled={busy}>
              Cancel
            </Button>
            <Button colorScheme="red" onClick={confirm} ml={3} isLoading={busy}>
              Delete with its {plural(n, "translation", "translations")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialogOverlay>
    </AlertDialog>
  );
}

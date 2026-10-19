import { useState } from "react";
import { Alert, AlertIcon, Box, Button, Flex, Heading, HStack, Icon, Text, useDisclosure } from "@chakra-ui/react";
import { FiPlus } from "react-icons/fi";
import { FaPenNib } from "react-icons/fa6";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../../../services/api";
import { useLocales } from "../../../shared/hooks/useLocales";
import { useNotify } from "../../../shared/hooks/useNotify";
import { PageLoader } from "../../../shared/ui/page-loader";
import type { Schema } from "../../../shared/types";
import { useMe } from "../../auth/me";
import { HistoryDrawer } from "../../history/HistoryDrawer";
import { RecentlyDeletedButton } from "../../history/RecentlyDeleted";
import { type ContentItem, useCollectionContent, useCollectionsList, useSingletonsList } from "../hooks";
import { ContentSelector } from "./ContentSelector";
import { ContentTable } from "./ContentTable";
import { EntryFormModal } from "./EntryFormModal";
import { hasLocalizedAttributes } from "./entryValues";
import { LocaleSwitcher } from "./LocaleSwitcher";
import { SingletonEditor } from "./SingletonEditor";
import { type Selection, selectionFromParams, selectionParams } from "./selection";

export function ContributionsPage() {
  const [params, setParams] = useSearchParams();
  const { collections, loading: collectionsLoading, error: collectionsError } = useCollectionsList();
  const { singletons, loading: singletonsLoading, error: singletonsError } = useSingletonsList(true);

  // The URL names the selection; otherwise the first collection (or single
  // type) is shown.
  const fallback: Selection | null = collections[0]?.schema.collectionName
    ? { kind: "collection", name: collections[0].schema.collectionName }
    : singletons[0]
      ? { kind: "singleton", name: singletons[0].name }
      : null;
  const selected = selectionFromParams(params) ?? fallback;
  const select = (s: Selection) => setParams(selectionParams(s));

  const schema = selected?.kind === "collection" ? collections.find((c) => c.schema.collectionName === selected.name)?.schema : undefined;
  const singleton = selected?.kind === "singleton" ? singletons.find((s) => s.name === selected.name) : undefined;
  const listsLoading = collectionsLoading || singletonsLoading;

  return (
    <Flex h="calc(100vh - 64px)">
      <ContentSelector
        collections={collections}
        singletons={singletons}
        showSingletons
        selected={selected}
        onSelect={select}
        collectionsLoading={collectionsLoading}
        singletonsLoading={singletonsLoading}
        singletonsError={singletonsError}
      />

      <Box flex="1" minW={0} p={{ base: 4, lg: 8 }} overflowY="auto">
        {collectionsError && (
          <Alert status="error" borderRadius="md" mb={6}>
            <AlertIcon />
            {collectionsError}
          </Alert>
        )}
        {schema && selected && <CollectionContent key={selected.name} collectionName={selected.name} schema={schema} />}
        {singleton && <SingletonEditor key={singleton.name} singleton={singleton} />}
        {!schema && !singleton && !listsLoading && (
          <Flex justify="center" align="center" h="100%">
            <Text color="gray.400">
              {selected ? `“${selected.name}” was not found.` : "Nothing to edit yet: create a collection or a single type in the content model."}
            </Text>
          </Flex>
        )}
      </Box>
    </Flex>
  );
}

function CollectionContent({ collectionName, schema }: { collectionName: string; schema: Schema }) {
  const navigate = useNavigate();
  const notify = useNotify();
  const { canWrite } = useMe();
  const { isOpen, onOpen, onClose } = useDisclosure();
  const localized = hasLocalizedAttributes(schema.attributes);
  const { locales, defaultLocale } = useLocales(localized);
  const translatable = localized && locales.length > 1 && !!defaultLocale;
  // The locale the list is read in; null is the default locale.
  const [listLocale, setListLocale] = useState<string | null>(null);
  const viewLocale = translatable && listLocale && listLocale !== defaultLocale?.code && locales.some((l) => l.code === listLocale) ? listLocale : null;
  const { content, loading, error, refetch, hasMore, loadMore } = useCollectionContent(collectionName, viewLocale);
  const [editingItem, setEditingItem] = useState<ContentItem | null>(null);
  // Bumped on every open, so the form starts from the entry it opens for.
  const [formKey, setFormKey] = useState(0);
  // The entry whose history is open.
  const [historyItem, setHistoryItem] = useState<ContentItem | null>(null);
  const base = `/api/collections/${encodeURIComponent(collectionName)}`;

  const openForm = (item: ContentItem | null) => {
    setEditingItem(item);
    setFormKey((k) => k + 1);
    onOpen();
  };

  const handleSaveItem = async (payload: { data: Partial<ContentItem> }, id?: string | number) => {
    try {
      await api(id ? "PUT" : "POST", id ? `${base}/${id}` : base, payload);
      notify.success(id ? "Entry updated" : "Entry created");
      refetch();
    } catch (err) {
      notify.failure(err, "Could not save the entry");
      throw err;
    }
  };

  const handleDelete = async (item: ContentItem) => {
    try {
      await api("DELETE", `${base}/${item.id}`);
      notify.success("Entry deleted");
      refetch();
    } catch (err) {
      notify.failure(err, "Could not delete the entry");
    }
  };

  // A restored version replaces the entry: an open form for it would be
  // stale, so it closes, and the list reloads.
  const handleRestored = () => {
    if (isOpen && editingItem && historyItem && String(editingItem.id) === String(historyItem.id)) onClose();
    refetch();
  };

  const handleTogglePublish = async (item: ContentItem) => {
    const action = item.status === "published" ? "unpublish" : "publish";
    try {
      await api("POST", `${base}/${item.id}/${action}`);
      refetch();
    } catch (err) {
      notify.failure(err, `Could not ${action}`);
    }
  };

  return (
    <>
      <Flex justify="space-between" align="center" mb={6} gap={4} wrap="wrap">
        <Box>
          <Heading size="lg">{schema.info?.displayName || schema.collectionName}</Heading>
          <Text color="gray.500">{loading && content.items.length === 0 ? "Loading entries…" : `${content.total} ${content.total === 1 ? "entry" : "entries"}, newest first`}</Text>
        </Box>
        {(canWrite || translatable) && (
        <HStack>
          {translatable && defaultLocale && (
            <LocaleSwitcher label="View in locale" locales={locales} value={viewLocale ?? defaultLocale.code} onChange={setListLocale} />
          )}
          {canWrite && (
          <>
          <RecentlyDeletedButton collection={collectionName} onRestored={refetch} />
          <Button bg="white" variant="outline" size="sm" leftIcon={<Icon as={FiPlus} />} onClick={() => openForm(null)}>
            Quick Add
          </Button>
          <Button colorScheme="purple" size="sm" leftIcon={<Icon as={FaPenNib} />} onClick={() => navigate(`/contrib/edit/${encodeURIComponent(collectionName)}`)}>
            Contrib
          </Button>
          </>
          )}
        </HStack>
        )}
      </Flex>
      {error && (
        <Alert status="error" borderRadius="md" mb={6}>
          <AlertIcon />
          {error}
        </Alert>
      )}
      {/* While entries reload after a change, the table stays on screen. */}
      {loading && content.items.length === 0 && !error && <PageLoader text="Fetching content..." />}
      {!error && (!loading || content.items.length > 0) && (
        <ContentTable
          schema={schema}
          items={content.items}
          canWrite={canWrite}
          locales={translatable ? locales : undefined}
          onAdd={() => openForm(null)}
          onEdit={openForm}
          onHistory={setHistoryItem}
          onTogglePublish={handleTogglePublish}
          onDelete={handleDelete}
        />
      )}
      {!error && hasMore && (
        <Flex justify="center" mt={4}>
          <Button size="sm" variant="outline" bg="white" onClick={loadMore} isLoading={loading}>
            Load more ({content.total - content.items.length} older)
          </Button>
        </Flex>
      )}
      <EntryFormModal
        key={`${collectionName}-${formKey}`}
        isOpen={isOpen}
        onClose={onClose}
        schema={schema}
        itemToEdit={editingItem}
        onSubmit={handleSaveItem}
        onShowHistory={canWrite ? setHistoryItem : undefined}
        collection={collectionName}
        locales={translatable ? locales : undefined}
        // An entry opened from a list read in a locale is edited in it.
        initialLocale={editingItem ? viewLocale : null}
        onTranslationSaved={refetch}
      />
      <HistoryDrawer
        target={historyItem ? { kind: "item", collection: collectionName, id: historyItem.id, label: entryLabel(schema, historyItem) } : null}
        onClose={() => setHistoryItem(null)}
        onRestored={handleRestored}
      />
    </>
  );
}

/** "Title (#12)" for an entry, or "#12" when it has no title. */
function entryLabel(schema: Schema, item: ContentItem): string {
  const title = ["title", "name"].map((k) => item[k]).find((v) => typeof v === "string" && v);
  return title ? `${title} (#${item.id})` : `${schema.info?.displayName || schema.collectionName} #${item.id}`;
}

import { useState } from "react";
import { Alert, AlertIcon, Box, Button, Flex, Heading, HStack, Icon, Text, useDisclosure } from "@chakra-ui/react";
import { FiPlus } from "react-icons/fi";
import { FaPenNib } from "react-icons/fa6";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api } from "../../../services/api";
import { useNotify } from "../../../shared/hooks/useNotify";
import { PageLoader } from "../../../shared/ui/page-loader";
import type { Schema } from "../../../shared/types";
import { useMe } from "../../auth/me";
import { ContentItem, useCollectionContent, useCollectionsList, useSingletonsList } from "../hooks";
import { ContentSelector } from "./ContentSelector";
import { ContentTable } from "./ContentTable";
import { EntryFormModal } from "./EntryFormModal";
import { SingletonEditor } from "./SingletonEditor";
import { Selection, selectionFromParams, selectionParams } from "./selection";

export function ContributionsPage() {
  const { isAdmin } = useMe();
  const [params, setParams] = useSearchParams();
  const { collections, loading: collectionsLoading, error: collectionsError } = useCollectionsList();
  const { singletons, loading: singletonsLoading, error: singletonsError } = useSingletonsList(isAdmin);

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
  const listsLoading = collectionsLoading || (isAdmin && singletonsLoading);

  return (
    <Flex h="calc(100vh - 64px)">
      <ContentSelector
        collections={collections}
        singletons={singletons}
        showSingletons={isAdmin}
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
  const { isOpen, onOpen, onClose } = useDisclosure();
  const { content, loading, error, refetch } = useCollectionContent(collectionName);
  const [editingItem, setEditingItem] = useState<ContentItem | null>(null);
  // Bumped on every open, so the form starts from the entry it opens for.
  const [formKey, setFormKey] = useState(0);
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
          <Text color="gray.500">{loading && content.items.length === 0 ? "Loading entries…" : `${content.items.length} entries found`}</Text>
        </Box>
        <HStack>
          <Button bg="white" variant="outline" size="sm" leftIcon={<Icon as={FiPlus} />} onClick={() => openForm(null)}>
            Quick Add
          </Button>
          <Button colorScheme="purple" size="sm" leftIcon={<Icon as={FaPenNib} />} onClick={() => navigate(`/contrib/edit/${encodeURIComponent(collectionName)}`)}>
            Contrib
          </Button>
        </HStack>
      </Flex>
      {error && (
        <Alert status="error" borderRadius="md" mb={6}>
          <AlertIcon />
          {error}
        </Alert>
      )}
      {/* While entries reload after a change, the table stays on screen. */}
      {loading && content.items.length === 0 && !error && <PageLoader text="Fetching content..." />}
      {!error && (!loading || content.items.length > 0) && <ContentTable schema={schema} items={content.items} onEdit={openForm} onTogglePublish={handleTogglePublish} />}
      <EntryFormModal key={`${collectionName}-${formKey}`} isOpen={isOpen} onClose={onClose} schema={schema} itemToEdit={editingItem} onSubmit={handleSaveItem} />
    </>
  );
}

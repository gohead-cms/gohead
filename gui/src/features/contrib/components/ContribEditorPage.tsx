import { useMemo, useState } from "react";
import {
  Accordion,
  AccordionButton,
  AccordionIcon,
  AccordionItem,
  AccordionPanel,
  Alert,
  AlertIcon,
  Box,
  Button,
  Flex,
  FormControl,
  Grid,
  GridItem,
  Heading,
  HStack,
  Icon,
  Input,
  Text,
  Textarea,
  VStack,
} from "@chakra-ui/react";
import { useNavigate, useParams } from "react-router-dom";
import { FiSave, FiX } from "react-icons/fi";
import { api } from "../../../services/api";
import { localeLabel, useLocales } from "../../../shared/hooks/useLocales";
import { useNotify } from "../../../shared/hooks/useNotify";
import { PageLoader } from "../../../shared/ui/page-loader";
import type { Attribute, Schema } from "../../../shared/types";
import { useCollectionSchema } from "../../collections/hook/useCollectionSchema";
import { EntryFields } from "./EntryFields";
import { hasLocalizedAttributes, initialValues, missingRequired, toEntryPayload } from "./entryValues";
import { LocaleSwitcher } from "./LocaleSwitcher";

type Field = { key: string; attr: Attribute };

// Places a collection's fields: the title on top, the first long text as the
// main editor, everything else in the sidebar.
function layoutFields(schema: Schema) {
  let title: Field | null = null;
  let main: Field | null = null;
  const sidebar: Record<string, Attribute> = {};
  for (const [key, attr] of Object.entries(schema.attributes)) {
    if (attr.type === "relation") continue;
    if (!title && key.toLowerCase() === "title") title = { key, attr };
    else if (!main && (attr.type === "richtext" || attr.type === "text")) main = { key, attr };
    else sidebar[key] = attr;
  }
  return { titleField: title as Field | null, mainField: main as Field | null, sidebarFields: sidebar };
}

export function ContributionEditorPage() {
  const { collectionName } = useParams<{ collectionName: string }>();
  const { schema, loading, error } = useCollectionSchema(collectionName);
  if (error) {
    return (
      <Box p={8}>
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          {error}
        </Alert>
      </Box>
    );
  }
  if (loading || !schema || !collectionName) return <PageLoader text="Loading editor..." />;
  // A new collection starts a new, empty form.
  return <ContributionEditor key={collectionName} collectionName={collectionName} schema={schema} />;
}

function ContributionEditor({ collectionName, schema }: { collectionName: string; schema: Schema }) {
  const navigate = useNavigate();
  const notify = useNotify();
  const { mainField, titleField, sidebarFields } = useMemo(() => layoutFields(schema), [schema]);
  const [formData, setFormData] = useState<Record<string, unknown>>(() => initialValues(schema.attributes));
  const [missing, setMissing] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const localized = hasLocalizedAttributes(schema.attributes);
  const { locales, defaultLocale } = useLocales(localized);

  const handleChange = (key: string, value: unknown) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async () => {
    const empty = missingRequired(schema.attributes, formData);
    setMissing(empty);
    if (empty.length > 0) return;
    setIsSubmitting(true);
    try {
      await api("POST", `/api/collections/${encodeURIComponent(collectionName)}`, { data: toEntryPayload(schema.attributes, formData) });
      notify.success("Entry saved");
      navigate("/contrib");
    } catch (err) {
      notify.failure(err, "Could not save the entry");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Box bg="gray.50" minH="calc(100vh - 64px)">
      <Flex as="header" position="sticky" top="64px" zIndex="docked" bg="white" px={8} py={3} borderBottomWidth="1px" justify="space-between" align="center">
        <Box>
          <Heading size="md">{schema.info?.displayName || collectionName}</Heading>
          <Text fontSize="sm" color="gray.500">
            Creative Editor
          </Text>
        </Box>
        <HStack>
          {localized && locales.length > 1 && defaultLocale && (
            // Entries are created in the default locale; translations are
            // added from the entry's editor once it exists.
            <LocaleSwitcher
              locales={locales}
              value={defaultLocale.code}
              onChange={() => {}}
              disabledReason={`New entries are written in ${localeLabel(defaultLocale)}. Once saved, open the entry from the list to translate it.`}
            />
          )}
          <Button variant="ghost" leftIcon={<Icon as={FiX} />} onClick={() => navigate("/contrib")}>
            Cancel
          </Button>
          <Button colorScheme="purple" leftIcon={<Icon as={FiSave} />} onClick={handleSubmit} isLoading={isSubmitting}>
            Save Entry
          </Button>
        </HStack>
      </Flex>
      <Grid templateColumns={{ base: "1fr", lg: "minmax(0, 3fr) minmax(0, 1fr)" }} gap={8} maxW="1200px" mx="auto" p={8}>
        <GridItem as={VStack} spacing={6} align="stretch">
          {missing.length > 0 && (
            <Alert status="warning" borderRadius="md">
              <AlertIcon />
              Required: {missing.join(", ")}
            </Alert>
          )}
          {titleField && (
            <FormControl isRequired={titleField.attr.required}>
              <Input
                aria-label={titleField.key}
                value={String(formData[titleField.key] ?? "")}
                onChange={(e) => handleChange(titleField.key, e.target.value)}
                placeholder="Your post title..."
                variant="unstyled"
                fontSize="4xl"
                fontWeight="bold"
              />
            </FormControl>
          )}
          {mainField && (
            <FormControl isRequired={mainField.attr.required}>
              <Textarea
                aria-label={mainField.key}
                value={String(formData[mainField.key] ?? "")}
                onChange={(e) => handleChange(mainField.key, e.target.value)}
                placeholder="Begin writing your content..."
                variant="unstyled"
                fontSize="lg"
                rows={20}
              />
            </FormControl>
          )}
        </GridItem>
        {Object.keys(sidebarFields).length > 0 && (
          <GridItem as={VStack} spacing={4} align="stretch" pos="sticky" top="150px" alignSelf="start">
            <Accordion allowToggle defaultIndex={[0]}>
              <AccordionItem borderWidth="0" bg="white" borderRadius="lg" boxShadow="sm">
                <AccordionButton>
                  <Box flex="1" textAlign="left" fontWeight="semibold">
                    Metadata
                  </Box>
                  <AccordionIcon />
                </AccordionButton>
                <AccordionPanel pb={4}>
                  <EntryFields attributes={sidebarFields} values={formData} onChange={handleChange} idPrefix="editor-" />
                </AccordionPanel>
              </AccordionItem>
            </Accordion>
          </GridItem>
        )}
      </Grid>
    </Box>
  );
}

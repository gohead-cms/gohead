import React, { useState, useEffect } from "react";
import {
  Box,
  Flex,
  Heading,
  Text,
  Button,
  VStack,
  Grid,
  GridItem,
  Input,
  Textarea,
  Switch,
  FormControl,
  FormLabel,
  useToast,
  HStack,
  Icon,
  Accordion,
  AccordionItem,
  AccordionButton,
  AccordionPanel,
  AccordionIcon,
} from "@chakra-ui/react";
import { useNavigate, useParams } from "react-router-dom";
import { FiSave, FiX } from "react-icons/fi";
import { PageLoader } from "../../../shared/ui/page-loader";
import { apiFetchWithAuth } from "../../../services/api"
import { Attribute } from "../../../shared/types";
import { useCollectionSchema } from "../../collections/hook/useCollectionSchema";


// Helper to render form fields in the sidebar

// Helper to render form fields in the sidebar
const renderSidebarField = (key: string, attr: Attribute, value: any, handleChange: any) => {
    const commonProps = {
      id: key,
      value: value || '',
      onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => handleChange(key, e.target.value),
      size: 'sm',
    };
  
    switch (attr.type) {
      case 'number':
        return <Input type="number" {...commonProps} />;
      case 'date':
        return <Input type="date" {...commonProps} />;
      case 'boolean':
        return (
          <Flex justify="space-between" align="center">
            <FormLabel htmlFor={key} mb="0" fontSize="sm">
              Enable
            </FormLabel>
            <Switch
              id={key}
              isChecked={value || false}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => handleChange(key, e.target.checked)}
            />
          </Flex>
        );
      default:
        return <Input type="text" {...commonProps} />;
    }
};

export function ContributionEditorPage() {
  const { collectionName } = useParams<{ collectionName: string }>();
  const navigate = useNavigate();
  const toast = useToast();

  const { schema, loading } = useCollectionSchema(collectionName!);
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [isSubmitting, setIsSubmitting] = useState(false);

  const [mainField, setMainField] = useState<{ key: string, attr: any } | null>(null);
  const [titleField, setTitleField] = useState<{ key: string, attr: any } | null>(null);
  const [sidebarFields, setSidebarFields] = useState<Record<string, any>>({});

  
  useEffect(() => {
    if (schema) {
      const attributes = schema.attributes;
      let main: { key: string, attr: Attribute } | null = null;
      let title: { key: string, attr: Attribute } | null = null;
      const sidebar: Record<string, Attribute> = {};
      const initialData: Record<string, any> = {};

      Object.entries(attributes).forEach(([key, rawAttr]) => {
        const attr = rawAttr as Attribute; // Add type assertion
        if (attr.type === 'relation') return;
        if (!title && key.toLowerCase() === 'title') {
          title = { key, attr };
        } else if (!main && (attr.type === 'richtext' || attr.type === 'text')) {
          main = { key, attr };
        } else {
          sidebar[key] = attr;
        }
        initialData[key] = attr.type === 'boolean' ? false : '';
      });
      setTitleField(title);
      setMainField(main);
      setSidebarFields(sidebar);
      setFormData(initialData);
    }
  }, [schema]);

  const handleChange = (key: string, value: any) => {
    setFormData(prev => ({ ...prev, [key]: value }));
  };

  const handleSubmit = async () => {
    setIsSubmitting(true);
    try {
        const response = await apiFetchWithAuth(`/api/collections/${collectionName}`, {
            method: 'POST',
            body: JSON.stringify(formData),
        });
        if (!response.ok) throw new Error("Failed to save entry.");
        toast({ title: "Entry Saved", status: "success", duration: 3000, isClosable: true });
        navigate('/contrib');
    } catch (error: any) {
        toast({ title: "Error", description: error.message, status: "error", duration: 5000, isClosable: true });
    } finally {
        setIsSubmitting(false);
    }
  };

  if (loading || !schema) return <PageLoader text="Loading editor..." />;

  return (
    <Box bg="gray.50" minH="calc(100vh - 64px)">
      <Flex as="header" position="sticky" top="64px" zIndex="docked" bg="white" px={8} py={3} borderBottomWidth="1px" justify="space-between" align="center">
        <Box>
            <Heading size="md">{schema.info?.displayName}</Heading>
            <Text fontSize="sm" color="gray.500">Creative Editor</Text>
        </Box>
        <HStack>
          <Button variant="ghost" leftIcon={<Icon as={FiX} />} onClick={() => navigate('/contrib')}>Cancel</Button>
          <Button colorScheme="purple" leftIcon={<Icon as={FiSave} />} onClick={handleSubmit} isLoading={isSubmitting}>Save Entry</Button>
        </HStack>
      </Flex>
      <Grid templateColumns={{ base: '1fr', lg: '3fr 1fr' }} gap={8} maxW="1200px" mx="auto" p={8}>
        <GridItem as={VStack} spacing={6} align="stretch">
          {titleField && (
            <FormControl isRequired={titleField.attr.required}>
              <Input value={formData[titleField.key] || ''} onChange={(e) => handleChange(titleField.key, e.target.value)} placeholder="Your post title..." variant="unstyled" fontSize="4xl" fontWeight="bold"/>
            </FormControl>
          )}
          {mainField && (
            <FormControl isRequired={mainField.attr.required}>
              <Textarea value={formData[mainField.key] || ''} onChange={(e) => handleChange(mainField.key, e.target.value)} placeholder="Begin writing your content..." variant="unstyled" fontSize="lg" rows={20}/>
            </FormControl>
          )}
        </GridItem>
        <GridItem as={VStack} spacing={4} align="stretch" pos="sticky" top="150px">
              <Accordion allowToggle defaultIndex={[0]}>
                <AccordionItem borderWidth="0" bg="white" borderRadius="lg" boxShadow="sm">
                  <AccordionButton>
                    <Box flex="1" textAlign="left" fontWeight="semibold">Metadata</Box>
                    <AccordionIcon />
                  </AccordionButton>
                  <AccordionPanel pb={4}>
                    <VStack spacing={4} align="stretch">
                      {Object.entries(sidebarFields).map(([key, attr]) => (
                        <FormControl key={key} isRequired={attr.required}>
                          <FormLabel htmlFor={key} fontSize="sm" fontWeight="medium">{key}</FormLabel>
                          {renderSidebarField(key, attr, formData[key], handleChange)}
                        </FormControl>
                      ))}
                    </VStack>
                  </AccordionPanel>
                </AccordionItem>
              </Accordion>
        </GridItem>
      </Grid>
    </Box>
  );
}


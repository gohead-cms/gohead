import React, { useEffect, useState } from "react";
import {
  Box,
  Heading,
  Text,
  Flex,
  Badge,
  Icon,
  Spinner,
  Button,
  IconButton,
  HStack,
} from "@chakra-ui/react";
import {
  FiDatabase,
  FiCheckCircle,
  FiLock,
  FiHash,
  FiMail,
  FiKey,
  FiToggleLeft,
  FiLink,
  FiEdit2,
  FiTrash2,
  FiPlus,
  FiSettings,
  FiArrowLeft,
  FiTrash,
  FiSave,
  FiEdit3,
} from "react-icons/fi";
import { apiFetchWithAuth } from "../../utils/api";

const typeIcons: Record<string, React.ElementType> = {
  text: FiHash,
  email: FiMail,
  password: FiKey,
  boolean: FiToggleLeft,
  relation: FiLink,
};

function getTypeIcon(type: string) {
  const IconComp = typeIcons[type] || FiHash;
  return <Icon as={IconComp} boxSize={4} mr={2} />;
}

interface Attribute {
  type: string;
  required?: boolean;
  enum?: string[];
  target?: string;
}

export interface Schema {
  attributes: Record<string, any>;
  collectionName?: string;
  info?: {
    displayName?: string;
    pluralName?: string;
    singularName?: string;
  };
}

export default function CollectionSchema({ collectionName }: { collectionName: string }) {
  const [schema, setSchema] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!collectionName) return;
    setLoading(true);
    apiFetchWithAuth(`/admin/collections/${collectionName}`)
      .then(async (res) => {
        if (!res.ok) throw new Error("Failed to fetch schema");
        const json = await res.json();
        setSchema(json.data.schema);
      })
      .finally(() => setLoading(false));
  }, [collectionName]);

  if (loading) return <Spinner />;
  if (!schema) return <Text color="gray.500">No schema found</Text>;

  return (
    <Box>
      {/* Header section */}
      <Flex align="center" mb={4} gap={4} justify="space-between">
        <HStack>
          <Heading size="xl">
            {schema.info?.displayName || schema.collectionName}
          </Heading>
          <Button
            size="sm"
            colorScheme="gray"
            variant="outline"
            fontWeight="normal"
          >
            <Icon as={FiEdit2} mr={2} /> Edit
          </Button>
        </HStack>
        <HStack>
          <Button
            colorScheme="blue"
            variant="outline"
            fontWeight="normal"
            size="sm"
          >
            <Icon as={FiPlus} mr={2} /> Add another field
          </Button>
          <Button
            colorScheme="gray"
            variant="outline"
            size="sm"
          >
            <Icon as={FiSave} mr={2} /> Save
          </Button>
        </HStack>
      </Flex>

      <Text color="gray.500" mb={4}>
        Build the data architecture of your content.
      </Text>

      {/* Config view button */}
      {/* <!--
      <Flex justify="flex-end" mb={4}>
        <Button size="sm" colorScheme="gray" variant="outline">
          <Icon as={FiSettings} mr={2} /> Configure the view
        </Button>
      </Flex> */}

      {/* Attributes Table */}
      <Box
        borderRadius="xl"
        boxShadow="sm"
        px={0}
        pt={0}
        pb={0}
        mb={6}
      >
        <Box overflowX="auto">
          <table style={{ width: "100%", borderCollapse: "separate", borderSpacing: 0 }}>
            <thead>
              <tr style={{ background: "#F6F6FB", textAlign: "left" }}>
                <th style={{ padding: "16px 24px", fontWeight: 700, fontSize: "14px" }}>NAME</th>
                <th style={{ padding: "16px 24px", fontWeight: 700, fontSize: "14px" }}>TYPE</th>
                <th style={{ padding: "16px 24px", fontWeight: 700, fontSize: "14px" }}>REQUIRED</th>
                <th style={{ padding: "16px 24px", fontWeight: 700, fontSize: "14px" }}>OPTIONS</th>
                <th style={{ width: 70 }}></th>
              </tr>
            </thead>
            <tbody>
              {Object.entries(schema.attributes).map(([attrName, attr], i) => {
                const attribute = attr as Attribute;
                return (
                  <tr key={attrName} style={{
                    background: i % 2 === 0 ? "#fff" : "#FAFAFA",
                    borderBottom: "1px solid #F3F3F6"
                  }}>
                    <td style={{ padding: "18px 24px" }}>
                      <Flex align="center">
                        {getTypeIcon(attribute.type)}
                        <Text fontWeight="bold">{attrName}</Text>
                      </Flex>
                    </td>
                    <td style={{ padding: "18px 24px" }}>
                      <Text color="gray.700">{attribute.type === "relation"
                        ? <>Relation with <i>{attribute.target?.split(".").pop()}</i></>
                        : attribute.type.charAt(0).toUpperCase() + attribute.type.slice(1)
                      }</Text>
                    </td>
                    <td style={{ padding: "18px 24px" }}>
                      {attribute.required ? (
                        <Icon as={FiCheckCircle} color="green.500" />
                      ) : (
                        <Icon as={FiLock} color="gray.300" />
                      )}
                    </td>
                    <td style={{ padding: "18px 24px" }}>
                      {attribute.enum
                        ? <Badge colorScheme="yellow">{attribute.enum.join(", ")}</Badge>
                        : <Text color="gray.400">—</Text>}
                    </td>
                    <td style={{ padding: "18px 24px" }}>
                      <HStack>
                      <IconButton
                          aria-label="Edit"
                          size="sm"
                          variant="ghost"
                          color="gray.500"
                        >
                        <FiEdit3 />
                       </IconButton>
                        <IconButton
                          aria-label="Edit"
                          size="sm"
                          variant="ghost"
                          color="gray.500"
                        >
                        <FiTrash2 />
                       </IconButton>
                      </HStack>
                    </td>
                  </tr>
                );
              })}
            </tbody>

          </table>
        </Box>
        {/* Add another field row */}
        <Box
          py={5}
          px={7}
          borderBottomRadius="xl"
          bg="#F6F6FB"
          textAlign="left"
          cursor="pointer"
          transition="background 0.1s"
          _hover={{ background: "#EBECFB" }}
        >
          <Button
            colorScheme="blue"
            variant="ghost"
            fontWeight="bold"
            size="sm"
          >
            <Icon as={FiPlus} mr={2} /> Add another field to this collection
          </Button>
        </Box>
      </Box>
    </Box>
  );
}

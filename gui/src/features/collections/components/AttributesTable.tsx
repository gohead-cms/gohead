import React from "react";
import {
  Box,
  Flex,
  Text,
  Icon,
  Badge,
  IconButton,
  HStack,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
} from "@chakra-ui/react";
import {
  FiCheckCircle,
  FiLock,
  FiHash,
  FiMail,
  FiKey,
  FiToggleLeft,
  FiLink,
  FiEdit3,
  FiTrash2,
} from "react-icons/fi";

const typeIcons: Record<string, React.ElementType> = {
  text: FiHash,
  email: FiMail,
  password: FiKey,
  boolean: FiToggleLeft,
  relation: FiLink,
};

function getTypeIcon(type: string) {
  const IconComp = typeIcons[type] || FiHash;
  return <Icon as={IconComp} boxSize={4} />;
}

export function AttributesTable({ attributes }: { attributes: Record<string, any> }) {
  return (
    <TableContainer borderWidth="1px" borderColor="gray.200" borderRadius="xl">
      <Table variant="simple">
        <Thead bg="gray.50">
          <Tr>
            <Th>Name</Th>
            <Th>Type</Th>
            <Th>Required</Th>
            <Th>Options</Th>
            <Th isNumeric></Th>
          </Tr>
        </Thead>
        <Tbody>
          {Object.entries(attributes).map(([attrName, attr]) => (
            <Tr key={attrName}>
              <Td>
                <HStack spacing={3}>
                  {getTypeIcon(attr.type)}
                  <Text fontWeight="bold">{attrName}</Text>
                </HStack>
              </Td>
              <Td>
                {attr.type === "relation" ? (
                  <Text as="i">Relation with {attr.target?.split(".").pop()}</Text>
                ) : (
                  <Text>{attr.type.charAt(0).toUpperCase() + attr.type.slice(1)}</Text>
                )}
              </Td>
              <Td>
                {attr.required ? (
                  <Icon as={FiCheckCircle} color="green.500" />
                ) : (
                  <Icon as={FiLock} color="gray.300" />
                )}
              </Td>
              <Td>
                {attr.enum ? (
                  <Badge colorScheme="yellow">{attr.enum.join(", ")}</Badge>
                ) : (
                  "—"
                )}
              </Td>
              <Td isNumeric>
                <HStack spacing={1} justify="flex-end">
                  <IconButton icon={<FiEdit3 />} aria-label="Edit" variant="ghost" />
                  <IconButton icon={<FiTrash2 />} aria-label="Delete" variant="ghost" />
                </HStack>
              </Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    </TableContainer>
  );
}
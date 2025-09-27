import React from "react";
import {
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
  IconButton,
  HStack,
  Badge,
  Box,
} from "@chakra-ui/react";
import { FiEdit, FiTrash2 } from "react-icons/fi";
import { Schema } from "../../../shared/types";
import { ContentItem } from "../hooks";

interface ContentTableProps {
  schema: Schema;
  items: ContentItem[];
}

// Function to get a displayable value from an item
const getDisplayValue = (value: any): string => {
    if (typeof value === 'object' && value !== null) {
        return JSON.stringify(value).substring(0, 30) + '...';
    }
    if (typeof value === 'string' && value.length > 50) {
        return value.substring(0, 50) + '...';
    }
    if (value === null || value === undefined) {
        return '–';
    }
    return String(value);
}

export function ContentTable({ schema, items }: ContentTableProps) {
  // Determine table headers from the schema, prioritizing common fields
  const getHeaders = () => {
    const attributes = Object.keys(schema.attributes);
    const preferredHeaders = ['id', 'title', 'name', 'slug', 'status'];
    let headers = attributes.filter(attr => preferredHeaders.includes(attr.toLowerCase()));
    if (headers.length < 4) {
        headers = [...headers, ...attributes.filter(attr => !headers.includes(attr))];
    }
    return headers.slice(0, 5); // Show a max of 5 columns for clarity
  };

  const headers = getHeaders();

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden">
      <TableContainer>
        <Table variant="simple">
          <Thead bg="gray.50">
            <Tr>
              {headers.map((header) => (
                <Th key={header}>{header}</Th>
              ))}
              <Th isNumeric>Actions</Th>
            </Tr>
          </Thead>
          <Tbody>
            {items.map((item) => (
              <Tr key={item.id}>
                {headers.map((header) => (
                  <Td key={`${item.id}-${header}`}>
                    {getDisplayValue(item[header])}
                  </Td>
                ))}
                <Td isNumeric>
                  <HStack spacing={2} justify="flex-end">
                    <IconButton
                      aria-label="Edit item"
                      icon={<FiEdit />}
                      size="sm"
                      variant="ghost"
                    />
                    <IconButton
                      aria-label="Delete item"
                      icon={<FiTrash2 />}
                      size="sm"
                      variant="ghost"
                      colorScheme="red"
                    />
                  </HStack>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </TableContainer>
    </Box>
  );
}

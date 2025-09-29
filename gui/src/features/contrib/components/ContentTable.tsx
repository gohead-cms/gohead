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
  Box,
} from "@chakra-ui/react";
import { FiEdit, FiTrash2 } from "react-icons/fi";
import { Schema } from "../../../shared/types";
import { ContentItem } from "../hooks";
import { EmptyState } from "./EmptyState";

interface ContentTableProps {
  schema: Schema;
  items: ContentItem[];
  onEdit: (item: ContentItem) => void; 
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

export function ContentTable({ schema, items, onEdit }: ContentTableProps) {
  // Determine table headers from the schema, prioritizing common fields
  const getHeaders = () => {
    const attributes = Object.keys(schema.attributes);
    // Let's add 'id' manually to our preferred list if it's not in the schema attributes
    const preferredHeaders = ['id', 'title', 'name', 'slug', 'status'];
    let headers = attributes.filter(attr => preferredHeaders.includes(attr.toLowerCase()));
    if (headers.length < 4) {
        headers = [...headers, ...attributes.filter(attr => !headers.includes(attr))];
    }
    // Ensure 'id' is always the first column if present
    const finalHeaders = ['id', ...headers.filter(h => h !== 'id')];
    return finalHeaders.slice(0, 5); // Show a max of 5 columns for clarity
  };

  const headers = getHeaders();

  if (!items || items.length === 0) {
    return <EmptyState />;
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
      <TableContainer>
        <Table variant="simple">
          <Thead bg="gray.50">
            <Tr>
              {headers.map((header) => (
                <Th key={header} textTransform="capitalize">{header.replace(/_/g, ' ')}</Th>
              ))}
              <Th isNumeric>Actions</Th>
            </Tr>
          </Thead>
          <Tbody>
            {items.map((item) => (
              <Tr key={item.id}>
                {headers.map((header) => (
                  <Td key={`${item.id}-${header}`}>
                    {/* FIX: Access the id directly, and other properties from the nested 'attributes' object */}
                    {getDisplayValue(header === 'id' ? item.id : item.attributes?.[header])}
                  </Td>
                ))}
                <Td isNumeric>
                  <HStack spacing={1} justify="flex-end">
                    <IconButton
                      aria-label="Edit item"
                      icon={<FiEdit />}
                      size="sm"
                      variant="ghost"
                      onClick={() => onEdit(item)} 
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


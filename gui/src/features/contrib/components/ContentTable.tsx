import { Table, Thead, Tbody, Tr, Th, Td, TableContainer, IconButton, HStack, Box, Badge, Button, Tooltip } from "@chakra-ui/react";
import { FiClock, FiEdit, FiTrash2, FiEye, FiEyeOff, FiInbox, FiPlus } from "react-icons/fi";
import type { Locale } from "../../../shared/hooks/useLocales";
import type { Schema } from "../../../shared/types";
import { ConfirmButton } from "../../../shared/ui/ConfirmButton";
import { EmptyState } from "../../../shared/ui/EmptyState";
import type { ContentItem } from "../hooks";
import { displayValue, tableColumns } from "./entryValues";

interface ContentTableProps {
  schema: Schema;
  items: ContentItem[];
  /** False hides the actions that change content (viewers). */
  canWrite: boolean;
  /** With localized attributes and several locales: a column of the locales each entry has. */
  locales?: Locale[];
  onAdd: () => void;
  onEdit: (item: ContentItem) => void;
  /** Opens the entry's history (editors and admins). */
  onHistory?: (item: ContentItem) => void;
  onTogglePublish: (item: ContentItem) => void;
  onDelete: (item: ContentItem) => Promise<void>;
}

export function ContentTable({ schema, items, canWrite, locales, onAdd, onEdit, onHistory, onTogglePublish, onDelete }: ContentTableProps) {
  const drafts = !!schema.options?.draftAndPublish;
  const headers = tableColumns(schema);

  if (items.length === 0) {
    return (
      <EmptyState
        icon={FiInbox}
        title="This collection is empty"
        action={
          canWrite && (
            <Button size="sm" colorScheme="purple" leftIcon={<FiPlus />} onClick={onAdd}>
              Add the first entry
            </Button>
          )
        }
      >
        {canWrite ? "Get started by creating your first entry." : "Nothing has been published here yet."}
      </EmptyState>
    );
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
      <TableContainer>
        <Table variant="simple" size="sm">
          <Thead bg="gray.50">
            <Tr>
              {headers.map((header) => (
                <Th key={header}>{header.replace(/_/g, " ")}</Th>
              ))}
              {locales && <Th>Locales</Th>}
              {drafts && <Th>Status</Th>}
              {canWrite && <Th isNumeric>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {items.map((item) => {
              const published = item.status === "published";
              return (
                <Tr key={item.id}>
                  {headers.map((header) => (
                    // Items are flat: attributes sit next to id.
                    <Td key={header}>{displayValue(item[header])}</Td>
                  ))}
                  {locales && (
                    <Td>
                      <EntryLocales codes={item.locales} locales={locales} />
                    </Td>
                  )}
                  {drafts && (
                    <Td>
                      <Badge colorScheme={published ? "green" : "gray"}>{item.status}</Badge>
                    </Td>
                  )}
                  {canWrite && (
                    <Td isNumeric>
                      <HStack spacing={1} justify="flex-end">
                        {drafts && (
                          <Tooltip label={published ? "Unpublish" : "Publish"}>
                            <IconButton
                              aria-label={published ? "Unpublish item" : "Publish item"}
                              icon={published ? <FiEyeOff /> : <FiEye />}
                              size="sm"
                              variant="ghost"
                              onClick={() => onTogglePublish(item)}
                            />
                          </Tooltip>
                        )}
                        <IconButton aria-label="Edit item" icon={<FiEdit />} size="sm" variant="ghost" onClick={() => onEdit(item)} />
                        {onHistory && (
                          <Tooltip label="History">
                            <IconButton aria-label="Item history" icon={<FiClock />} size="sm" variant="ghost" onClick={() => onHistory(item)} />
                          </Tooltip>
                        )}
                        <ConfirmButton
                          trigger={<IconButton aria-label="Delete item" icon={<FiTrash2 />} size="sm" variant="ghost" colorScheme="red" />}
                          title="Delete this entry?"
                          body={`Entry #${item.id} will be deleted. You can bring it back later from "Recently deleted".`}
                          onConfirm={() => onDelete(item)}
                        />
                      </HStack>
                    </Td>
                  )}
                </Tr>
              );
            })}
          </Tbody>
        </Table>
      </TableContainer>
    </Box>
  );
}

/** The locales an entry has content in: the default, then its translations. */
function EntryLocales({ codes, locales }: { codes: string[] | undefined; locales: Locale[] }) {
  const has = new Set(codes ?? []);
  return (
    <HStack spacing={1} flexWrap="nowrap">
      {locales.map((l) => {
        const present = l.isDefault || has.has(l.code);
        return (
          <Tooltip key={l.code} label={l.isDefault ? `${l.code}: default locale` : present ? `${l.code}: translated` : `${l.code}: not translated (falls back)`}>
            <Badge
              fontSize="2xs"
              colorScheme={l.isDefault ? "purple" : present ? "green" : "gray"}
              variant={present ? "subtle" : "outline"}
              opacity={present ? 1 : 0.6}
              textTransform="none"
            >
              {l.code}
            </Badge>
          </Tooltip>
        );
      })}
    </HStack>
  );
}

import { Table, IconButton, HStack, Box, Badge, Button } from "@chakra-ui/react";
import { Tooltip } from "@/components/ui/tooltip";
import { FiClock, FiEdit, FiTrash2, FiEye, FiEyeOff, FiInbox, FiPlus } from "react-icons/fi";
import type { Locale } from "../../../shared/hooks/useLocales";
import type { Schema } from "../../../shared/types";
import { ConfirmButton } from "../../../shared/ui/ConfirmButton";
import { EmptyState } from "../../../shared/ui/EmptyState";
import type { ContentItem } from "../hooks";
import { PUBLISHED_LOCKED } from "../../history/history";
import type { EntryPermissions } from "./entryPermissions";
import { displayValue, tableColumns } from "./entryValues";

interface ContentTableProps {
  schema: Schema;
  items: ContentItem[];
  /** What the role may do here: the actions it cannot are hidden. */
  perms: EntryPermissions;
  /** With localized attributes and several locales: a column of the locales each entry has. */
  locales?: Locale[];
  onAdd: () => void;
  onEdit: (item: ContentItem) => void;
  /** Opens the entry's history (roles that can update the type). */
  onHistory?: (item: ContentItem) => void;
  onTogglePublish: (item: ContentItem) => void;
  onDelete: (item: ContentItem) => Promise<void>;
}

export function ContentTable({ schema, items, perms, locales, onAdd, onEdit, onHistory, onTogglePublish, onDelete }: ContentTableProps) {
  const drafts = !!schema.options?.draftAndPublish;
  const headers = tableColumns(schema);
  const canPublish = drafts && perms.publish;
  const actions = canPublish || perms.update || perms.delete || !!onHistory;

  if (items.length === 0) {
    return (
      <EmptyState
        icon={FiInbox}
        title="This collection is empty"
        action={
          perms.create && (
            <Button size="xs" colorPalette="purple" onClick={onAdd}>
              <FiPlus />
              Add the first entry
            </Button>
          )
        }
      >
        {perms.create ? "Get started by creating your first entry." : "Nothing has been published here yet."}
      </EmptyState>
    );
  }

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
      <Table.ScrollArea>
        <Table.Root size="sm">
          <Table.Header bg="gray.50">
            <Table.Row>
              {headers.map((header) => (
                <Table.ColumnHeader key={header}>{header.replace(/_/g, " ")}</Table.ColumnHeader>
              ))}
              {locales && <Table.ColumnHeader>Locales</Table.ColumnHeader>}
              {drafts && <Table.ColumnHeader>Status</Table.ColumnHeader>}
              {actions && <Table.ColumnHeader textAlign="end">Actions</Table.ColumnHeader>}
            </Table.Row>
          </Table.Header>
          <Table.Body>
            {items.map((item) => {
              const published = item.status === "published";
              return (
                <Table.Row key={item.id}>
                  {headers.map((header) => (
                    // Items are flat: attributes sit next to id.
                    <Table.Cell key={header}>{displayValue(item[header])}</Table.Cell>
                  ))}
                  {locales && (
                    <Table.Cell>
                      <EntryLocales codes={item.locales} locales={locales} />
                    </Table.Cell>
                  )}
                  {drafts && (
                    <Table.Cell>
                      <Badge colorPalette={published ? "green" : "gray"}>{item.status}</Badge>
                    </Table.Cell>
                  )}
                  {actions && (
                    <Table.Cell textAlign="end">
                      <HStack gap={1} justify="flex-end">
                        {canPublish && (
                          <Tooltip content={published ? "Unpublish" : "Publish"}>
                            <IconButton aria-label={published ? "Unpublish item" : "Publish item"} size="xs" variant="ghost" onClick={() => onTogglePublish(item)}>
                              {published ? <FiEyeOff /> : <FiEye />}
                            </IconButton>
                          </Tooltip>
                        )}
                        {perms.update &&
                          (drafts && published && !perms.publish ? (
                            // Changing published content needs publish (the server says 403).
                            <Tooltip content={PUBLISHED_LOCKED} showArrow>
                              <span tabIndex={0} aria-label={PUBLISHED_LOCKED}>
                                <IconButton aria-label="Edit item" size="xs" variant="ghost" disabled>
                                  <FiEdit />
                                </IconButton>
                              </span>
                            </Tooltip>
                          ) : (
                            <IconButton aria-label="Edit item" size="xs" variant="ghost" onClick={() => onEdit(item)}>
                              <FiEdit />
                            </IconButton>
                          ))}
                        {onHistory && (
                          <Tooltip content="History">
                            <IconButton aria-label="Item history" size="xs" variant="ghost" onClick={() => onHistory(item)}>
                              <FiClock />
                            </IconButton>
                          </Tooltip>
                        )}
                        {perms.delete && (
                          <ConfirmButton
                            trigger={
                              <IconButton aria-label="Delete item" size="xs" variant="ghost" colorPalette="red">
                                <FiTrash2 />
                              </IconButton>
                            }
                            title="Delete this entry?"
                            body={
                              perms.update
                                ? `Entry #${item.id} will be deleted. You can bring it back later from "Recently deleted".`
                                : `Entry #${item.id} will be deleted. A role that can edit this collection can bring it back from "Recently deleted".`
                            }
                            onConfirm={() => onDelete(item)}
                          />
                        )}
                      </HStack>
                    </Table.Cell>
                  )}
                </Table.Row>
              );
            })}
          </Table.Body>
        </Table.Root>
      </Table.ScrollArea>
    </Box>
  );
}

/** The locales an entry has content in: the default, then its translations. */
function EntryLocales({ codes, locales }: { codes: string[] | undefined; locales: Locale[] }) {
  const has = new Set(codes ?? []);
  return (
    <HStack gap={1} flexWrap="nowrap">
      {locales.map((l) => {
        const present = l.isDefault || has.has(l.code);
        return (
          <Tooltip key={l.code} content={l.isDefault ? `${l.code}: default locale` : present ? `${l.code}: translated` : `${l.code}: not translated (falls back)`}>
            <Badge
              fontSize="2xs"
              colorPalette={l.isDefault ? "purple" : present ? "green" : "gray"}
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

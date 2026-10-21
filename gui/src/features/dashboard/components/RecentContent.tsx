import { Link as RouterLink } from "react-router-dom";
import { Button, Link, Table, Text } from "@chakra-ui/react";
import { FiFileText } from "react-icons/fi";
import { EmptyState } from "../../../shared/ui/EmptyState";
import { formatTime } from "../../agents/components/runFormat";
import { contribLink } from "../../contrib/components/selection";
import type { RecentItem } from "../overview";
import { timeAgo } from "../overview";

/** The entries changed last, each linking to its collection in the editor. */
export function RecentContent({ items }: { items: RecentItem[] }) {
  if (items.length === 0) {
    return (
      <EmptyState
        icon={FiFileText}
        title="No content yet"
        action={
          <Button asChild size="xs" colorPalette="purple">
            <RouterLink to="/contrib">Open the content editor</RouterLink>
          </Button>
        }
      >
        Entries you create or edit will show up here.
      </EmptyState>
    );
  }
  return (
    <Table.ScrollArea>
      <Table.Root size="sm">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>Entry</Table.ColumnHeader>
            <Table.ColumnHeader>Collection</Table.ColumnHeader>
            <Table.ColumnHeader>Updated</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {items.map((it) => {
            const label = it.title || `Entry #${it.id}`;
            return (
              <Table.Row key={it.id}>
                <Table.Cell maxW="280px">
                  {it.collection ? (
                    <Link asChild color="purple.600" display="block" truncate title={label}>
                      <RouterLink to={contribLink({ kind: "collection", name: it.collection })}>{label}</RouterLink>
                    </Link>
                  ) : (
                    <Text truncate>{label}</Text>
                  )}
                </Table.Cell>
                <Table.Cell>{it.collection || "–"}</Table.Cell>
                <Table.Cell title={formatTime(it.updatedAt)} color="gray.600">
                  {timeAgo(it.updatedAt)}
                </Table.Cell>
              </Table.Row>
            );
          })}
        </Table.Body>
      </Table.Root>
    </Table.ScrollArea>
  );
}

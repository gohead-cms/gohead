import { Link as RouterLink } from "react-router-dom";
import { Button, Link, Table, TableContainer, Tbody, Td, Text, Th, Thead, Tr } from "@chakra-ui/react";
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
          <Button as={RouterLink} to="/contrib" size="sm" colorScheme="purple">
            Open the content editor
          </Button>
        }
      >
        Entries you create or edit will show up here.
      </EmptyState>
    );
  }
  return (
    <TableContainer>
      <Table size="sm">
        <Thead>
          <Tr>
            <Th>Entry</Th>
            <Th>Collection</Th>
            <Th>Updated</Th>
          </Tr>
        </Thead>
        <Tbody>
          {items.map((it) => {
            const label = it.title || `Entry #${it.id}`;
            return (
              <Tr key={it.id}>
                <Td maxW="280px">
                  {it.collection ? (
                    <Link as={RouterLink} to={contribLink({ kind: "collection", name: it.collection })} color="purple.600" display="block" isTruncated title={label}>
                      {label}
                    </Link>
                  ) : (
                    <Text isTruncated>{label}</Text>
                  )}
                </Td>
                <Td>{it.collection || "–"}</Td>
                <Td title={formatTime(it.updatedAt)} color="gray.600">
                  {timeAgo(it.updatedAt)}
                </Td>
              </Tr>
            );
          })}
        </Tbody>
      </Table>
    </TableContainer>
  );
}

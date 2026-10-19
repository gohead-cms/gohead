import { Link as RouterLink } from "react-router-dom";
import {
  Alert,
  AlertDescription,
  AlertIcon,
  Badge,
  Box,
  Button,
  Center,
  Flex,
  HStack,
  IconButton,
  Link,
  Spinner,
  Table,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
} from "@chakra-ui/react";
import { FiEdit2, FiLayers, FiPlus, FiTrash2 } from "react-icons/fi";
import { api } from "../../../services/api";
import { useApiQuery } from "../../../shared/hooks/useApiQuery";
import { useNotify } from "../../../shared/hooks/useNotify";
import { ConfirmButton } from "../../../shared/ui/ConfirmButton";
import { EmptyState } from "../../../shared/ui/EmptyState";
import { loadSummaries, serverMessage, TypeSummary } from "../api";
import { deletePath, KIND_DESCRIPTIONS, KIND_LABELS, ModelKind } from "../schema";

/** The types of one kind, with create, edit and delete actions. */
export function TypeTable({ kind }: { kind: ModelKind }) {
  const { data, loading, error, refetch } = useApiQuery(`model:${kind}`, () => loadSummaries(kind));
  const notify = useNotify();
  const { singular } = KIND_LABELS[kind];

  const remove = async (name: string) => {
    try {
      await api("DELETE", deletePath(kind, name));
      notify.success(`Deleted ${singular} "${name}"`);
      refetch();
    } catch (err) {
      notify.failure(new Error(serverMessage(err)), `Could not delete "${name}"`);
    }
  };

  const newButton = (
    <Button as={RouterLink} to={`/model/${kind}/new`} leftIcon={<FiPlus />} colorScheme="teal" size="sm">
      New {singular}
    </Button>
  );

  return (
    <Box>
      <Flex justify="space-between" align="center" gap={4} mb={4} wrap="wrap">
        <Text color="gray.600">{KIND_DESCRIPTIONS[kind]}</Text>
        {newButton}
      </Flex>
      {error ? (
        <Alert status="error" borderRadius="md">
          <AlertIcon />
          <AlertDescription flex="1">{error}</AlertDescription>
          <Button size="sm" onClick={refetch}>
            Retry
          </Button>
        </Alert>
      ) : loading && !data ? (
        <Center py={10}>
          <Spinner />
        </Center>
      ) : !data || data.length === 0 ? (
        <EmptyState icon={FiLayers} title={`No ${singular} yet`} action={newButton}>
          {KIND_DESCRIPTIONS[kind]}
        </EmptyState>
      ) : (
        <Box bg="white" borderWidth="1px" borderRadius="lg" overflowX="auto">
          <Table size="sm">
            <Thead>
              <Tr>
                <Th>Name</Th>
                <Th>Description</Th>
                <Th isNumeric>Attributes</Th>
                {kind === "collections" && <Th>Workflow</Th>}
                <Th textAlign="right">Actions</Th>
              </Tr>
            </Thead>
            <Tbody>
              {data.map((t) => (
                <TypeRow key={t.name} kind={kind} type={t} onDelete={() => remove(t.name)} />
              ))}
            </Tbody>
          </Table>
        </Box>
      )}
    </Box>
  );
}

function TypeRow({ kind, type, onDelete }: { kind: ModelKind; type: TypeSummary; onDelete: () => Promise<void> }) {
  const editPath = `/model/${kind}/${encodeURIComponent(type.name)}`;
  const { singular } = KIND_LABELS[kind];
  return (
    <Tr>
      <Td fontWeight="medium">
        <Link as={RouterLink} to={editPath} color="teal.600">
          {type.name}
        </Link>
      </Td>
      <Td color="gray.600" maxW="sm">
        <Text noOfLines={1}>{type.description || "—"}</Text>
      </Td>
      <Td isNumeric>{type.attributeCount}</Td>
      {kind === "collections" && (
        <Td>
          {type.draftAndPublish ? <Badge colorScheme="purple">Draft &amp; publish</Badge> : <Text color="gray.400">—</Text>}
        </Td>
      )}
      <Td>
        <HStack justify="flex-end" spacing={1}>
          <Button as={RouterLink} to={editPath} size="xs" variant="ghost" leftIcon={<FiEdit2 />} aria-label={`Edit ${type.name}`}>
            Edit
          </Button>
          <ConfirmButton
            trigger={<IconButton aria-label={`Delete ${type.name}`} icon={<FiTrash2 />} size="xs" variant="ghost" colorScheme="red" />}
            title={`Delete ${singular} "${type.name}"?`}
            body={deleteWarning(kind)}
            onConfirm={onDelete}
          />
        </HStack>
      </Td>
    </Tr>
  );
}

function deleteWarning(kind: ModelKind): string {
  switch (kind) {
    case "collections":
      return "The collection and all its items are deleted. This cannot be undone.";
    case "components":
      return "The component definition is deleted. A component still used by a type cannot be deleted.";
    case "singletons":
      return "The singleton and its content are deleted. This cannot be undone.";
  }
}

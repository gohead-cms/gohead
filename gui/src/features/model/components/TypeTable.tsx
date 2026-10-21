import { Link as RouterLink } from "react-router-dom";
import {
  Alert,
  Badge,
  Box,
  Button,
  Center,
  Flex,
  HStack,
  Icon,
  IconButton,
  Link,
  Spinner,
  Table,
  Text,
} from "@chakra-ui/react";
import { Tooltip } from "@/components/ui/tooltip";
import { FiEdit2, FiGlobe, FiLayers, FiPlus, FiTrash2 } from "react-icons/fi";
import { api } from "../../../services/api";
import { useApiQuery } from "../../../shared/hooks/useApiQuery";
import { useNotify } from "../../../shared/hooks/useNotify";
import { ConfirmButton } from "../../../shared/ui/ConfirmButton";
import { EmptyState } from "../../../shared/ui/EmptyState";
import { loadSummaries, serverMessage } from "../api";
import type { TypeSummary } from "../api";
import { deletePath, KIND_DESCRIPTIONS, KIND_LABELS } from "../schema";
import type { ModelKind } from "../schema";

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
    <Button asChild colorPalette="teal" size="xs">
      <RouterLink to={`/model/${kind}/new`}>
        <FiPlus />
        New {singular}
      </RouterLink>
    </Button>
  );

  return (
    <Box>
      <Flex justify="space-between" align="center" gap={4} mb={4} wrap="wrap">
        <Text color="gray.600">{KIND_DESCRIPTIONS[kind]}</Text>
        {newButton}
      </Flex>
      {error ? (
        <Alert.Root status="error" borderRadius="md" alignItems="center">
          <Alert.Indicator />
          <Alert.Description flex="1">{error}</Alert.Description>
          <Button size="xs" variant="subtle" onClick={refetch}>
            Retry
          </Button>
        </Alert.Root>
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
          <Table.Root size="sm">
            <Table.Header>
              <Table.Row>
                <Table.ColumnHeader>Name</Table.ColumnHeader>
                <Table.ColumnHeader>Description</Table.ColumnHeader>
                <Table.ColumnHeader textAlign="end">Attributes</Table.ColumnHeader>
                {kind === "collections" && <Table.ColumnHeader>Workflow</Table.ColumnHeader>}
                <Table.ColumnHeader textAlign="right">Actions</Table.ColumnHeader>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {data.map((t) => (
                <TypeRow key={t.name} kind={kind} type={t} onDelete={() => remove(t.name)} />
              ))}
            </Table.Body>
          </Table.Root>
        </Box>
      )}
    </Box>
  );
}

function TypeRow({ kind, type, onDelete }: { kind: ModelKind; type: TypeSummary; onDelete: () => Promise<void> }) {
  const editPath = `/model/${kind}/${encodeURIComponent(type.name)}`;
  const { singular } = KIND_LABELS[kind];
  return (
    <Table.Row>
      <Table.Cell fontWeight="medium">
        <HStack gap={2}>
          <Link asChild color="teal.600">
            <RouterLink to={editPath}>{type.name}</RouterLink>
          </Link>
          {type.localizedCount > 0 && (
            <Tooltip showArrow content={`${type.localizedCount} localized ${type.localizedCount === 1 ? "attribute" : "attributes"}: a value per locale`}>
              <Badge colorPalette="purple" variant="subtle" display="inline-flex" alignItems="center" gap={1} aria-label="Localized">
                <Icon as={FiGlobe} />
                {type.localizedCount}
              </Badge>
            </Tooltip>
          )}
        </HStack>
      </Table.Cell>
      <Table.Cell color="gray.600" maxW="sm">
        <Text lineClamp={1}>{type.description || "—"}</Text>
      </Table.Cell>
      <Table.Cell textAlign="end">{type.attributeCount}</Table.Cell>
      {kind === "collections" && (
        <Table.Cell>
          {type.draftAndPublish ? <Badge colorPalette="purple">Draft &amp; publish</Badge> : <Text color="gray.400">—</Text>}
        </Table.Cell>
      )}
      <Table.Cell>
        <HStack justify="flex-end" gap={1}>
          <Button asChild size="2xs" variant="ghost" aria-label={`Edit ${type.name}`}>
            <RouterLink to={editPath}>
              <FiEdit2 />
              Edit
            </RouterLink>
          </Button>
          <ConfirmButton
            trigger={
              <IconButton aria-label={`Delete ${type.name}`} size="2xs" variant="ghost" colorPalette="red">
                <FiTrash2 />
              </IconButton>
            }
            title={`Delete ${singular} "${type.name}"?`}
            body={deleteWarning(kind)}
            onConfirm={onDelete}
          />
        </HStack>
      </Table.Cell>
    </Table.Row>
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

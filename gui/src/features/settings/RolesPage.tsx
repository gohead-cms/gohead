import { useState } from "react";
import {
  Alert,
  AlertDescription,
  AlertIcon,
  Badge,
  Box,
  Button,
  HStack,
  IconButton,
  Skeleton,
  Stack,
  Table,
  TableContainer,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tooltip,
  Tr,
} from "@chakra-ui/react";
import { FiEdit2, FiEye, FiPlus, FiShield, FiTrash2 } from "react-icons/fi";
import { api, ApiError } from "../../services/api";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { RoleBadge } from "./components/RoleBadge";
import { RoleFormDrawer } from "./components/RoleFormDrawer";
import { permissionSummary, type Role, roleHelp, useRoles } from "./components/roles";

/** Editing state: closed, creating (null) or showing/editing a role. */
type Editing = { role: Role | null; n: number } | undefined;

function plural(n: number, one: string, many: string) {
  return `${n} ${n === 1 ? one : many}`;
}

/** "2 users and 1 API token" from a 409's details, or null. */
function inUse(err: unknown): string | null {
  if (!(err instanceof ApiError) || err.status !== 409) return null;
  const d = err.details as { users?: number; tokens?: number } | undefined;
  if (!d || typeof d !== "object") return null;
  const parts = [];
  if (d.users) parts.push(plural(d.users, "user", "users"));
  if (d.tokens) parts.push(plural(d.tokens, "API token", "API tokens"));
  return parts.length ? parts.join(" and ") : null;
}

export function RolesPage() {
  const notify = useNotify();
  const [editing, setEditing] = useState<Editing>(undefined);
  const { data, loading, error, refetch } = useRoles();
  const roles = data ?? [];
  const custom = roles.filter((r) => !r.builtin);

  const open = (role: Role | null) => setEditing((e) => ({ role, n: (e?.n ?? 0) + 1 }));

  const remove = async (r: Role) => {
    try {
      await api("DELETE", `/admin/roles/${encodeURIComponent(r.name)}`);
      notify.success(`Role ${r.name} deleted`);
      refetch();
    } catch (err) {
      const users = inUse(err);
      if (users) notify.warning(`Role ${r.name} is still in use`, `It is given to ${users}. Give them another role first, then delete it.`);
      else notify.failure(err, "Could not delete the role");
    }
  };

  const newButton = (
    <Button leftIcon={<FiPlus />} colorScheme="blue" onClick={() => open(null)}>
      New role
    </Button>
  );

  return (
    <Page>
      <PageHeader
        title="Roles"
        description="What users and API tokens may do with content: built-in roles, and custom roles with permissions per content type."
        actions={newButton}
      />

      {error && (
        <Alert status="error" borderRadius="md" mb={4}>
          <AlertIcon />
          <AlertDescription flex="1">{error}</AlertDescription>
          <Button size="sm" onClick={refetch}>
            Retry
          </Button>
        </Alert>
      )}

      {loading && !data ? (
        <Stack bg="white" borderWidth="1px" borderRadius="lg" p={4}>
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} h="32px" />
          ))}
        </Stack>
      ) : data ? (
        <Stack spacing={6}>
          <Box bg="white" borderWidth="1px" borderRadius="lg" overflow="hidden">
            <TableContainer whiteSpace="normal">
              <Table size="sm">
                <Thead>
                  <Tr>
                    <Th>Role</Th>
                    <Th>Permissions</Th>
                    <Th textAlign="right">Actions</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {roles.map((r) => (
                    <Tr key={r.name}>
                      <Td maxW="280px">
                        <HStack spacing={2}>
                          <RoleBadge role={r.name} />
                          {r.builtin && (
                            <Badge variant="outline" colorScheme="gray" fontSize="2xs">
                              built-in
                            </Badge>
                          )}
                        </HStack>
                        <Text fontSize="xs" color="gray.500" mt={1} noOfLines={2} title={roleHelp(r)}>
                          {roleHelp(r)}
                        </Text>
                      </Td>
                      <Td fontSize="xs" color="gray.700">
                        {r.name === "admin" ? (
                          <Text>Everything, plus the admin area</Text>
                        ) : (
                          <Text noOfLines={2} title={permissionSummary(r.permissions)}>
                            {permissionSummary(r.permissions)}
                          </Text>
                        )}
                      </Td>
                      <Td>
                        <HStack spacing={1} justify="flex-end">
                          {r.builtin ? (
                            <Tooltip label="View permissions" hasArrow>
                              <IconButton aria-label={`View ${r.name}`} icon={<FiEye />} size="sm" variant="ghost" onClick={() => open(r)} />
                            </Tooltip>
                          ) : (
                            <>
                              <Tooltip label="Edit" hasArrow>
                                <IconButton aria-label={`Edit ${r.name}`} icon={<FiEdit2 />} size="sm" variant="ghost" onClick={() => open(r)} />
                              </Tooltip>
                              <ConfirmButton
                                title={`Delete the ${r.name} role?`}
                                body="Only a role no user or API token has can be deleted. This cannot be undone."
                                onConfirm={() => remove(r)}
                                trigger={
                                  <IconButton aria-label={`Delete ${r.name}`} icon={<FiTrash2 />} size="sm" variant="ghost" colorScheme="red" />
                                }
                              />
                            </>
                          )}
                        </HStack>
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </TableContainer>
          </Box>
          {custom.length === 0 && (
            <EmptyState icon={FiShield} title="No custom roles yet" action={newButton}>
              Create a role to give someone access to some content types only, or to let them write drafts without publishing.
            </EmptyState>
          )}
        </Stack>
      ) : null}

      {editing && (
        <RoleFormDrawer
          key={editing.n}
          isOpen
          role={editing.role}
          onClose={() => setEditing(undefined)}
          onSaved={(saved, created) => {
            notify.success(created ? `Role ${saved.name} created` : `Role ${saved.name} saved`);
            setEditing(undefined);
            refetch();
          }}
        />
      )}
    </Page>
  );
}

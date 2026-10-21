import { useState } from "react";
import { Alert, Badge, Box, Button, HStack, IconButton, Skeleton, Stack, Table, Text } from "@chakra-ui/react";
import { Tooltip } from "@/components/ui/tooltip";
import { FiEdit2, FiEye, FiGlobe, FiPlus, FiShield, FiTrash2 } from "react-icons/fi";
import { api, ApiError } from "../../services/api";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { RoleBadge } from "./components/RoleBadge";
import { RoleFormDrawer } from "./components/RoleFormDrawer";
import { isPublicRole, permissionSummary, publicAccessSummary, type Role, roleHelp, useRoles } from "./components/roles";

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
  const all = data ?? [];
  // The public role (anonymous requests) has its own section.
  const roles = all.filter((r) => !isPublicRole(r));
  const publicRole = all.find(isPublicRole);
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
    <Button colorPalette="blue" onClick={() => open(null)}>
      <FiPlus />
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
        <Alert.Root status="error" borderRadius="md" mb={4} alignItems="center">
          <Alert.Indicator />
          <Alert.Content flex="1">
            <Alert.Description>{error}</Alert.Description>
          </Alert.Content>
          <Button size="xs" variant="subtle" onClick={refetch}>
            Retry
          </Button>
        </Alert.Root>
      )}

      {loading && !data ? (
        <Stack bg="white" borderWidth="1px" borderRadius="lg" p={4}>
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} h="32px" />
          ))}
        </Stack>
      ) : data ? (
        <Stack gap={6}>
          <Box bg="white" borderWidth="1px" borderRadius="lg" overflow="hidden">
            <Table.ScrollArea whiteSpace="normal">
              <Table.Root size="sm">
                <Table.Header>
                  <Table.Row>
                    <Table.ColumnHeader>Role</Table.ColumnHeader>
                    <Table.ColumnHeader>Permissions</Table.ColumnHeader>
                    <Table.ColumnHeader textAlign="right">Actions</Table.ColumnHeader>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {roles.map((r) => (
                    <Table.Row key={r.name}>
                      <Table.Cell maxW="280px">
                        <HStack gap={2}>
                          <RoleBadge role={r.name} />
                          {r.builtin && (
                            <Badge variant="outline" colorPalette="gray" fontSize="2xs">
                              built-in
                            </Badge>
                          )}
                        </HStack>
                        <Text fontSize="xs" color="gray.500" mt={1} lineClamp={2} title={roleHelp(r)}>
                          {roleHelp(r)}
                        </Text>
                      </Table.Cell>
                      <Table.Cell fontSize="xs" color="gray.700">
                        {r.name === "admin" ? (
                          <Text>Everything, plus the admin area</Text>
                        ) : (
                          <Text lineClamp={2} title={permissionSummary(r.permissions)}>
                            {permissionSummary(r.permissions)}
                          </Text>
                        )}
                      </Table.Cell>
                      <Table.Cell>
                        <HStack gap={1} justify="flex-end">
                          {r.builtin ? (
                            <Tooltip content="View permissions" showArrow>
                              <IconButton aria-label={`View ${r.name}`} size="xs" variant="ghost" onClick={() => open(r)}>
                                <FiEye />
                              </IconButton>
                            </Tooltip>
                          ) : (
                            <>
                              <Tooltip content="Edit" showArrow>
                                <IconButton aria-label={`Edit ${r.name}`} size="xs" variant="ghost" onClick={() => open(r)}>
                                  <FiEdit2 />
                                </IconButton>
                              </Tooltip>
                              <ConfirmButton
                                title={`Delete the ${r.name} role?`}
                                body="Only a role no user or API token has can be deleted. This cannot be undone."
                                onConfirm={() => remove(r)}
                                trigger={
                                  <IconButton aria-label={`Delete ${r.name}`} size="xs" variant="ghost" colorPalette="red">
                                    <FiTrash2 />
                                  </IconButton>
                                }
                              />
                            </>
                          )}
                        </HStack>
                      </Table.Cell>
                    </Table.Row>
                  ))}
                </Table.Body>
              </Table.Root>
            </Table.ScrollArea>
          </Box>
          {custom.length === 0 && (
            <EmptyState icon={FiShield} title="No custom roles yet" action={newButton}>
              Create a role to give someone access to some content types only, or to let them write drafts without publishing.
            </EmptyState>
          )}
          {publicRole && <PublicAccess role={publicRole} onEdit={() => open(publicRole)} />}
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

/** The public role: what anonymous requests (no token) may read. */
function PublicAccess({ role, onEdit }: { role: Role; onEdit: () => void }) {
  const readable = publicAccessSummary(role.permissions);
  const closed = readable === null;
  return (
    <Box bg="white" borderWidth="1px" borderRadius="lg" p={4} data-testid="public-role">
      <HStack gap={4} align="flex-start">
        <Box color="orange.500" pt={1}>
          <FiGlobe />
        </Box>
        <Box flex="1" minW={0}>
          <HStack gap={2} flexWrap="wrap">
            <Text fontWeight="semibold">Public (anonymous visitors)</Text>
            <RoleBadge role={role.name} />
            <Badge variant="outline" colorPalette={closed ? "gray" : "orange"} fontSize="2xs">
              {closed ? "closed" : "open to anyone"}
            </Badge>
          </HStack>
          <Text fontSize="xs" color="gray.500" mt={1}>
            {roleHelp(role)}. It cannot be given to users or API tokens.
          </Text>
          <Text fontSize="sm" color="gray.700" mt={2}>
            {closed ? (
              <>
                <Text as="span" fontWeight="semibold">
                  No access
                </Text>
                : API calls without a token are refused.
              </>
            ) : (
              <>
                Anyone, without a token, can read the published content of{" "}
                <Text as="span" fontWeight="semibold">
                  {readable}
                </Text>
                . Drafts stay private.
              </>
            )}
          </Text>
        </Box>
        <Tooltip content="Edit public access" showArrow>
          <IconButton aria-label={`Edit ${role.name}`} size="xs" variant="ghost" onClick={onEdit}>
            <FiEdit2 />
          </IconButton>
        </Tooltip>
      </HStack>
    </Box>
  );
}

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
import { FiEdit2, FiPlus, FiTrash2, FiUsers } from "react-icons/fi";
import { api } from "../../services/api";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { useMe } from "../auth/me";
import { RoleBadge } from "./components/RoleBadge";
import { UserFormModal } from "./components/UserFormModal";
import { optTime, userCreatedAt, userRoleName } from "./components/settingsFormat";
import type { AdminUser } from "./components/types";

const SELF_DELETE_REASON = "You cannot delete your own account. Ask another admin if it must go.";

/** Editing state: closed, creating (null) or editing a user. */
type Editing = { user: AdminUser | null } | undefined;

export function UsersPage() {
  const { me } = useMe();
  const notify = useNotify();
  const [editing, setEditing] = useState<Editing>(undefined);
  const { data, loading, error, refetch } = useApiQuery("/admin/users", async () => {
    const res = await getJSON<{ users: AdminUser[] }>("/admin/users", "Failed to load users");
    return res.data?.users ?? [];
  });
  const users = data ?? [];
  const isSelf = (u: AdminUser) => me?.kind === "user" && (me.id === u.ID || me.username === u.username);

  const remove = async (u: AdminUser) => {
    try {
      await api("DELETE", `/admin/users/${u.ID}`);
      notify.success(`User ${u.username} deleted`);
      refetch();
    } catch (err) {
      notify.failure(err, "Could not delete the user");
    }
  };

  const newButton = (
    <Button leftIcon={<FiPlus />} colorScheme="blue" onClick={() => setEditing({ user: null })}>
      New user
    </Button>
  );

  return (
    <Page>
      <PageHeader
        title="Users"
        description="People who can sign in to this admin. Self-registration is off by default (auth.registration), so this is where accounts are created."
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
      ) : data && users.length === 0 ? (
        <EmptyState icon={FiUsers} title="No users yet" action={newButton}>
          Create an account for each person who edits content here.
        </EmptyState>
      ) : data ? (
        <Box bg="white" borderWidth="1px" borderRadius="lg" overflow="hidden">
          <TableContainer>
            <Table size="sm">
              <Thead>
                <Tr>
                  <Th>Username</Th>
                  <Th>Email</Th>
                  <Th>Role</Th>
                  <Th>Created</Th>
                  <Th textAlign="right">Actions</Th>
                </Tr>
              </Thead>
              <Tbody>
                {users.map((u) => {
                  const self = isSelf(u);
                  return (
                    <Tr key={u.ID}>
                      <Td fontWeight="medium">
                        <HStack spacing={2}>
                          <Text>{u.username}</Text>
                          {self && <Badge colorScheme="green">you</Badge>}
                        </HStack>
                      </Td>
                      <Td>{u.email}</Td>
                      <Td>
                        <RoleBadge role={userRoleName(u)} />
                      </Td>
                      <Td color="gray.600">{optTime(userCreatedAt(u))}</Td>
                      <Td>
                        <HStack spacing={1} justify="flex-end">
                          <IconButton
                            aria-label={`Edit ${u.username}`}
                            icon={<FiEdit2 />}
                            size="sm"
                            variant="ghost"
                            onClick={() => setEditing({ user: u })}
                          />
                          {self ? (
                            <Tooltip label={SELF_DELETE_REASON} hasArrow>
                              {/* A disabled button gets no hover events: the span shows the reason. */}
                              <span tabIndex={0} aria-label={SELF_DELETE_REASON}>
                                <IconButton aria-label={`Delete ${u.username}`} icon={<FiTrash2 />} size="sm" variant="ghost" isDisabled />
                              </span>
                            </Tooltip>
                          ) : (
                            <ConfirmButton
                              title={`Delete ${u.username}?`}
                              body={`${u.username} will no longer be able to sign in. Content they created stays. This cannot be undone.`}
                              onConfirm={() => remove(u)}
                              trigger={
                                <IconButton
                                  aria-label={`Delete ${u.username}`}
                                  icon={<FiTrash2 />}
                                  size="sm"
                                  variant="ghost"
                                  colorScheme="red"
                                />
                              }
                            />
                          )}
                        </HStack>
                      </Td>
                    </Tr>
                  );
                })}
              </Tbody>
            </Table>
          </TableContainer>
        </Box>
      ) : null}

      {editing && (
        <UserFormModal
          key={editing.user?.ID ?? "new"}
          isOpen
          user={editing.user}
          isSelf={editing.user ? isSelf(editing.user) : false}
          onClose={() => setEditing(undefined)}
          onSaved={(msg) => {
            notify.success(msg);
            setEditing(undefined);
            refetch();
          }}
        />
      )}
    </Page>
  );
}

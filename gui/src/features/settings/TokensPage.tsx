import { useState } from "react";
import { Alert, Badge, Box, Button, Code, IconButton, Skeleton, Stack, Table, Text } from "@chakra-ui/react";
import { FiKey, FiPlus, FiSlash } from "react-icons/fi";
import { api } from "../../services/api";
import { getJSON, useApiQuery } from "../../shared/hooks/useApiQuery";
import { useNotify } from "../../shared/hooks/useNotify";
import { ConfirmButton } from "../../shared/ui/ConfirmButton";
import { EmptyState } from "../../shared/ui/EmptyState";
import { Page, PageHeader } from "../../shared/ui/PageHeader";
import { RoleBadge } from "./components/RoleBadge";
import { SecretDialog } from "./components/SecretDialog";
import { TokenFormModal } from "./components/TokenFormModal";
import { optTime, tokenStatusColor } from "./components/settingsFormat";
import type { ApiToken, CreatedApiToken } from "./components/types";

export function TokensPage() {
  const notify = useNotify();
  const [creating, setCreating] = useState(false);
  const [formKey, setFormKey] = useState(0);
  const [created, setCreated] = useState<CreatedApiToken | null>(null);
  const { data, loading, error, refetch } = useApiQuery("/admin/tokens", async () => {
    const res = await getJSON<ApiToken[]>("/admin/tokens", "Failed to load API tokens");
    return Array.isArray(res.data) ? res.data : [];
  });
  const tokens = data ?? [];

  const openForm = () => {
    setFormKey((k) => k + 1);
    setCreating(true);
  };

  const revoke = async (t: ApiToken) => {
    try {
      await api("DELETE", `/admin/tokens/${t.id}`);
      notify.success(`Token ${t.name} revoked`);
      refetch();
    } catch (err) {
      notify.failure(err, "Could not revoke the token");
    }
  };

  const newButton = (
    <Button colorPalette="blue" onClick={openForm}>
      <FiPlus />
      New token
    </Button>
  );

  return (
    <Page>
      <PageHeader
        title="API tokens"
        description="Long-lived keys for front-ends, CI jobs and integrations (Authorization: Bearer ght_…). A token acts with the permissions of its role; give it the lowest role that works."
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
      ) : data && tokens.length === 0 ? (
        <EmptyState icon={FiKey} title="No API tokens" action={newButton}>
          Create a viewer token for a website that reads content, or an editor token for a CI job that writes it.
        </EmptyState>
      ) : data ? (
        <Box bg="white" borderWidth="1px" borderRadius="lg" overflow="hidden">
          <Table.ScrollArea whiteSpace="normal">
            <Table.Root size="sm">
              <Table.Header>
                <Table.Row>
                  <Table.ColumnHeader>Name</Table.ColumnHeader>
                  <Table.ColumnHeader>Role</Table.ColumnHeader>
                  <Table.ColumnHeader>Status</Table.ColumnHeader>
                  <Table.ColumnHeader>Created</Table.ColumnHeader>
                  <Table.ColumnHeader>Last used</Table.ColumnHeader>
                  <Table.ColumnHeader>Expires</Table.ColumnHeader>
                  <Table.ColumnHeader textAlign="right">Actions</Table.ColumnHeader>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {tokens.map((t) => {
                  const active = t.status === "active";
                  return (
                    <Table.Row key={t.id} opacity={active ? 1 : 0.65}>
                      <Table.Cell maxW="240px">
                        <Text fontWeight="medium" lineClamp={1} title={t.name}>
                          {t.name}
                        </Text>
                        <Code fontSize="xs" title="The token starts with these characters">
                          {t.prefix}…
                        </Code>
                        {t.description && (
                          <Text fontSize="xs" color="gray.500" lineClamp={1} title={t.description}>
                            {t.description}
                          </Text>
                        )}
                      </Table.Cell>
                      <Table.Cell>
                        <RoleBadge role={t.role} />
                      </Table.Cell>
                      <Table.Cell>
                        <Badge colorPalette={tokenStatusColor[t.status] ?? "gray"}>{t.status}</Badge>
                      </Table.Cell>
                      <Table.Cell color="gray.600">
                        <Text>{optTime(t.createdAt)}</Text>
                        {t.createdBy && (
                          <Text fontSize="xs" color="gray.500">
                            by {t.createdBy}
                          </Text>
                        )}
                      </Table.Cell>
                      <Table.Cell color="gray.600">{optTime(t.lastUsedAt, "Never")}</Table.Cell>
                      <Table.Cell color="gray.600">
                        {t.revokedAt ? `Revoked ${optTime(t.revokedAt)}` : optTime(t.expiresAt, "Never")}
                      </Table.Cell>
                      <Table.Cell textAlign="right">
                        {active && (
                          <ConfirmButton
                            title={`Revoke ${t.name}?`}
                            body="Anything using this token stops working immediately. This cannot be undone: create a new token to restore access."
                            confirmLabel="Revoke"
                            onConfirm={() => revoke(t)}
                            trigger={
                              <IconButton aria-label={`Revoke ${t.name}`} size="xs" variant="ghost" colorPalette="red">
                                <FiSlash />
                              </IconButton>
                            }
                          />
                        )}
                      </Table.Cell>
                    </Table.Row>
                  );
                })}
              </Table.Body>
            </Table.Root>
          </Table.ScrollArea>
        </Box>
      ) : null}

      <TokenFormModal
        key={formKey}
        isOpen={creating}
        onClose={() => setCreating(false)}
        onCreated={(t) => {
          setCreating(false);
          setCreated(t);
          refetch();
        }}
      />

      <SecretDialog
        title={created ? `Token "${created.name}" created` : "Token created"}
        label="API token"
        secret={created?.token ?? null}
        onClose={() => setCreated(null)}
      >
        <Text fontSize="sm" color="gray.600">
          Send it as <Code fontSize="xs">Authorization: Bearer &lt;token&gt;</Code>. It acts as a <strong>{created?.role}</strong>.
        </Text>
      </SecretDialog>
    </Page>
  );
}

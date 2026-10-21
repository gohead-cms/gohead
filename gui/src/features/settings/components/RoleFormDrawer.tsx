import { type FormEvent, useState } from "react";
import { Alert, Badge, Box, Button, Code, Flex, HStack, Input, Skeleton, Stack, Text, VStack } from "@chakra-ui/react";
import { DrawerBody, DrawerCloseTrigger, DrawerContent, DrawerFooter, DrawerHeader, DrawerRoot, DrawerTitle } from "@/components/ui/drawer";
import { Field } from "@/components/ui/field";
import { api } from "../../../services/api";
import { useApiQuery } from "../../../shared/hooks/useApiQuery";
import { errorMessage } from "../../../shared/utils/errors";
import { ANY_TYPE, type Permissions, PUBLIC_ACTIONS, singletonName } from "../../auth/permissions";
import { loadSummaries } from "../../model/api";
import { type MatrixValue, PermissionMatrix } from "./PermissionMatrix";
import { isPublicRole, type Role, roleHelp, roleNameError } from "./roles";

/** The content types a role can be given permissions on. */
function useContentTypes() {
  return useApiQuery("role-content-types", async () => {
    const [collections, singletons] = await Promise.all([loadSummaries("collections"), loadSummaries("singletons")]);
    return {
      collections: collections.map((c) => c.name).sort(),
      singletons: singletons.map((s) => s.name).sort(),
    };
  });
}

function toMatrix(p: Permissions | undefined): MatrixValue {
  const content: Record<string, string[]> = {};
  for (const [k, v] of Object.entries(p?.content ?? {})) content[k] = [...(v ?? [])];
  return { content, media: [...(p?.media ?? [])] };
}

/**
 * Creates (role = null), edits or shows (built-in roles, read-only) a role.
 * The public role (anonymous requests) can only be given read permissions.
 * Mount it with a key per role so its fields start from that role.
 */
export function RoleFormDrawer({
  isOpen,
  role,
  onClose,
  onSaved,
}: {
  isOpen: boolean;
  role: Role | null;
  onClose: () => void;
  onSaved: (saved: Role, created: boolean) => void;
}) {
  const creating = role === null;
  const readOnly = !!role?.builtin;
  const isPublic = !!role && isPublicRole(role);
  const [name, setName] = useState(role?.name ?? "");
  const [description, setDescription] = useState(role?.description ?? "");
  const [matrix, setMatrix] = useState<MatrixValue>(() => toMatrix(role?.permissions ?? { content: { [ANY_TYPE]: [] }, media: [] }));
  const [submitted, setSubmitted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);
  const types = useContentTypes();

  const known = new Set([ANY_TYPE, ...(types.data?.collections ?? []), ...(types.data?.singletons ?? []).map((s) => `singleton:${s}`)]);
  // Types deleted since the role was saved: the server rejects them.
  const stale = types.data ? Object.keys(matrix.content).filter((k) => !known.has(k)) : [];

  const nameError = creating ? roleNameError(name) : null;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (readOnly) return;
    setSubmitted(true);
    if (nameError) return;
    const content: Record<string, string[]> = {};
    for (const [k, v] of Object.entries(matrix.content)) {
      if (stale.includes(k)) continue;
      if (k === ANY_TYPE && v.length === 0) continue;
      content[k] = v;
    }
    const permissions = { content, media: matrix.media };
    setSaving(true);
    setServerError(null);
    try {
      const res = role
        ? await api<Role>("PUT", `/admin/roles/${encodeURIComponent(role.name)}`, { description: description.trim(), permissions })
        : await api<Role>("POST", "/admin/roles", {
            name,
            description: description.trim(),
            permissions,
          });
      onSaved(res.data, creating);
    } catch (err) {
      setServerError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const title = creating ? "New role" : readOnly ? `Role ${role.name}` : isPublic ? "Edit public access" : `Edit ${role.name}`;

  return (
    <DrawerRoot
      open={isOpen}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      size="xl"
      closeOnInteractOutside={!saving}
    >
      <DrawerContent>
        <Flex asChild direction="column" h="full">
          <form onSubmit={submit} noValidate>
            <DrawerCloseTrigger disabled={saving} />
            <DrawerHeader borderBottomWidth="1px">
              <HStack gap={3}>
                <DrawerTitle flex="none">{title}</DrawerTitle>
                {readOnly && <Badge colorPalette="purple">built-in</Badge>}
                {isPublic && <Badge colorPalette="orange">anonymous visitors</Badge>}
              </HStack>
            </DrawerHeader>
            <DrawerBody>
              <VStack gap={5} align="stretch" py={2}>
                {serverError && (
                  <Alert.Root status="error" borderRadius="md">
                    <Alert.Indicator />
                    <Alert.Content>
                      <Alert.Description>{serverError}</Alert.Description>
                    </Alert.Content>
                  </Alert.Root>
                )}
                {readOnly && (
                  <Alert.Root status="info" borderRadius="md">
                    <Alert.Indicator />
                    <Alert.Content>
                      <Alert.Description fontSize="sm">
                        {roleHelp(role)}. Built-in roles cannot be changed or deleted: create a custom role for other needs.
                        {role.name === "admin" && " Admins also manage the content model, agents, users, tokens and settings."}
                      </Alert.Description>
                    </Alert.Content>
                  </Alert.Root>
                )}
                {isPublic && (
                  <Alert.Root status="warning" borderRadius="md">
                    <Alert.Indicator />
                    <Alert.Content>
                      <Alert.Description fontSize="sm">
                        Anything granted here is readable by anyone on the internet, without a token (published content only, never drafts).
                        With nothing granted, API calls without a token are refused.
                      </Alert.Description>
                    </Alert.Content>
                  </Alert.Root>
                )}
                {!readOnly && (
                  <>
                    <Field
                      label="Name"
                      required={creating}
                      invalid={submitted && !!nameError}
                      errorText={nameError}
                      helperText={
                        submitted && nameError
                          ? undefined
                          : creating
                            ? "Lowercase letters, digits, - and _, starting with a letter. It cannot be changed later."
                            : isPublic
                              ? "The role of requests without a token. It cannot be renamed, deleted or given to users and API tokens."
                              : "Roles cannot be renamed: accounts and API tokens refer to them by name."
                      }
                    >
                      <Input
                        value={name}
                        onChange={(e) => setName(e.target.value.toLowerCase())}
                        placeholder="writer, reviewer…"
                        readOnly={!creating}
                        autoFocus={creating}
                        autoComplete="off"
                      />
                    </Field>
                    <Field label="Description">
                      <Input
                        value={description}
                        onChange={(e) => setDescription(e.target.value)}
                        placeholder="Who it is for, what it allows"
                      />
                    </Field>
                  </>
                )}

                <Box as="fieldset" minW={0}>
                  <Text as="legend" fontSize="sm" fontWeight="medium" mb={2}>
                    Permissions
                  </Text>
                  {isPublic ? (
                    <Text fontSize="sm" color="gray.500" mb={3}>
                      The types anonymous visitors may read, one by one. Files used by that content are returned with it; the media library
                      itself, types with password attributes and the admin area stay private.
                    </Text>
                  ) : (
                    !readOnly && (
                      <Text fontSize="sm" color="gray.500" mb={3}>
                        What accounts and tokens with this role may do with content. Drafts are visible to roles that can update or publish
                        a type. The admin area (content model, agents, users, settings) stays admin-only.
                      </Text>
                    )
                  )}
                  {types.error && (
                    <Alert.Root status="warning" borderRadius="md" mb={3} alignItems="center">
                      <Alert.Indicator />
                      <Alert.Content flex="1">
                        <Alert.Description fontSize="sm">
                          Could not load the content types ({types.error}): only “All other types” can be set.
                        </Alert.Description>
                      </Alert.Content>
                      <Button size="2xs" variant="subtle" onClick={types.refetch}>
                        Retry
                      </Button>
                    </Alert.Root>
                  )}
                  {stale.length > 0 && !readOnly && (
                    <Alert.Root status="warning" borderRadius="md" mb={3}>
                      <Alert.Indicator />
                      <Alert.Content>
                        <Alert.Description fontSize="sm">
                          {stale.map((k, i) => (
                            <span key={k}>
                              {i > 0 && ", "}
                              <Code fontSize="xs">{singletonName(k) ?? k}</Code>
                            </span>
                          ))}{" "}
                          no longer {stale.length === 1 ? "exists" : "exist"}:{" "}
                          {stale.length === 1 ? "its settings are" : "their settings are"} dropped when you save.
                        </Alert.Description>
                      </Alert.Content>
                    </Alert.Root>
                  )}
                  {types.loading && !types.data ? (
                    <Stack>
                      {[0, 1, 2].map((i) => (
                        <Skeleton key={i} h="32px" />
                      ))}
                    </Stack>
                  ) : (
                    <PermissionMatrix
                      value={matrix}
                      onChange={setMatrix}
                      collections={types.data?.collections ?? []}
                      singletons={types.data?.singletons ?? []}
                      isReadOnly={readOnly}
                      actions={isPublic ? PUBLIC_ACTIONS : undefined}
                      explicitOnly={isPublic}
                    />
                  )}
                </Box>
              </VStack>
            </DrawerBody>
            <DrawerFooter borderTopWidth="1px" gap={3}>
              <Button variant="subtle" onClick={onClose} disabled={saving}>
                {readOnly ? "Close" : "Cancel"}
              </Button>
              {!readOnly && (
                <Button type="submit" colorPalette="blue" loading={saving}>
                  {creating ? "Create role" : "Save changes"}
                </Button>
              )}
            </DrawerFooter>
          </form>
        </Flex>
      </DrawerContent>
    </DrawerRoot>
  );
}

import { type FormEvent, useState } from "react";
import { Alert, Box, Button, Code, HStack, Icon, Input, Link, NativeSelect, SimpleGrid, Text, VStack, chakra } from "@chakra-ui/react";
import { FiExternalLink, FiGithub, FiServer, FiZap } from "react-icons/fi";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { api } from "../../services/api";
import { errorMessage } from "../../shared/utils/errors";
import { type Connection, type ConnectionAuth, type ConnectionInput, type ConnectionPreset, nameError, slugify } from "./types";

const PRESET_ICONS: Record<string, typeof FiServer> = {
  github: FiGithub,
  zapier: FiZap,
};

/** "custom": a server that isn't in the catalog. */
const CUSTOM = "custom";

function ServiceChoice({ presets, onChoose }: { presets: ConnectionPreset[]; onChoose: (id: string) => void }) {
  const choices = [
    ...presets.map((p) => ({
      id: p.id,
      label: p.label,
      description: p.description,
    })),
    {
      id: CUSTOM,
      label: "Other MCP server",
      description: "Any server that speaks the Model Context Protocol over HTTP: your own, or a service's.",
    },
  ];
  return (
    <SimpleGrid columns={{ base: 1, md: 3 }} gap={3}>
      {choices.map((c) => (
        <chakra.button
          key={c.id}
          type="button"
          textAlign="start"
          borderWidth="1px"
          borderRadius="md"
          p={4}
          bg="white"
          _hover={{ borderColor: "purple.400", bg: "purple.50" }}
          onClick={() => onChoose(c.id)}
          aria-label={`Connect ${c.label}`}
        >
          <HStack gap={2} mb={2}>
            <Icon as={PRESET_ICONS[c.id] ?? FiServer} color="purple.500" />
            <Text fontWeight="semibold">{c.label}</Text>
          </HStack>
          <Text fontSize="sm" color="fg.muted">
            {c.description}
          </Text>
        </chakra.button>
      ))}
    </SimpleGrid>
  );
}

/**
 * Connects an app (connection = null) or edits a connection. Mount it with a
 * key per connection so its fields start from that connection.
 */
export function ConnectionFormDialog({
  connection,
  presets,
  encrypted,
  onClose,
  onSaved,
}: {
  connection: Connection | null;
  presets: ConnectionPreset[];
  /** Tokens are encrypted at rest (jwt_secret is set). */
  encrypted: boolean;
  onClose: () => void;
  onSaved: (saved: Connection, created: boolean, warning?: string) => void;
}) {
  const creating = connection === null;
  const [service, setService] = useState<string | null>(creating ? null : connection.preset || CUSTOM);
  const preset = presets.find((p) => p.id === service);
  const [label, setLabel] = useState(connection?.label ?? "");
  const [name, setName] = useState(connection?.name ?? "");
  const [nameTouched, setNameTouched] = useState(!creating);
  const [url, setUrl] = useState(connection?.url ?? "");
  const [auth, setAuth] = useState<ConnectionAuth>(connection?.authType ?? "bearer");
  const [headerName, setHeaderName] = useState(connection?.headerName ?? "");
  const [token, setToken] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const choose = (id: string) => {
    setService(id);
    const p = presets.find((x) => x.id === id);
    if (p) {
      setLabel(p.label);
      setName(p.id);
      setUrl(p.url);
      setAuth(p.authType);
      setHeaderName(p.headerName ?? "");
    } else {
      setLabel("");
      setName("");
      setUrl("");
      setAuth("bearer");
      setHeaderName("");
    }
    setNameTouched(Boolean(p));
  };

  const custom = service === CUSTOM;
  const effectiveName = nameTouched ? name : slugify(label);
  const errors = {
    label: label.trim() ? null : "A name is required.",
    name: creating ? nameError(effectiveName) : null,
    url: !custom ? null : !/^https?:\/\/\S+$/.test(url.trim()) ? "An http(s) URL." : null,
    headerName: custom && auth === "header" && !/^[A-Za-z0-9-]+$/.test(headerName.trim()) ? "An HTTP header name." : null,
    token:
      auth === "none"
        ? null
        : creating && !token.trim()
          ? "Paste the token."
          : !creating &&
              connection.hasToken &&
              !token.trim() &&
              (url.trim() !== connection.url || auth !== connection.authType || headerName.trim() !== (connection.headerName ?? ""))
            ? "Enter the token again to change where it is sent."
            : null,
  };
  const invalid = Object.values(errors).some(Boolean);

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    if (invalid) return;
    const body: ConnectionInput = { label: label.trim() };
    if (creating) {
      body.name = effectiveName;
      if (preset) body.preset = preset.id;
    }
    if (custom) {
      body.url = url.trim();
      body.authType = auth;
      body.headerName = auth === "header" ? headerName.trim() : "";
    }
    if (token.trim()) body.token = token.trim();
    setSaving(true);
    setServerError(null);
    try {
      const res = creating
        ? await api<Connection>("POST", "/admin/connections", body)
        : await api<Connection>("PUT", `/admin/connections/${connection.name}`, body);
      const warning = typeof res.meta?.warning === "string" ? res.meta.warning : undefined;
      onSaved(res.data, creating, warning);
    } catch (err) {
      setServerError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const title = creating ? (service ? `Connect ${preset?.label ?? "an MCP server"}` : "Connect an app") : `Edit ${connection.label}`;
  const tokenLabel = preset?.tokenLabel ?? (auth === "header" ? `Value of ${headerName.trim() || "the header"}` : "Token");

  return (
    <DialogRoot
      open
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      size={service ? "md" : "xl"}
      closeOnInteractOutside={!saving}
    >
      <DialogContent>
        {!service ? (
          <>
            <DialogHeader>
              <DialogTitle>{title}</DialogTitle>
            </DialogHeader>
            <DialogBody pb={6}>
              <Text fontSize="sm" color="fg.muted" mb={4}>
                Agents get the app's tools in their tool picker. Calls that may change something wait for an admin's approval.
              </Text>
              <ServiceChoice presets={presets} onChoose={choose} />
            </DialogBody>
          </>
        ) : (
          <form onSubmit={submit} noValidate>
            <DialogHeader>
              <DialogTitle>{title}</DialogTitle>
            </DialogHeader>
            <DialogBody>
              <VStack gap={4} align="stretch">
                {serverError && (
                  <Alert.Root role="alert" status="error" borderRadius="md">
                    <Alert.Indicator />
                    <Alert.Content>
                      <Alert.Description>{serverError}</Alert.Description>
                    </Alert.Content>
                  </Alert.Root>
                )}
                {preset && (
                  <Text fontSize="sm" color="fg.muted">
                    {preset.description}{" "}
                    {preset.docsUrl && (
                      <Link href={preset.docsUrl} target="_blank" rel="noreferrer" color="purple.600">
                        Learn more <FiExternalLink />
                      </Link>
                    )}
                  </Text>
                )}
                <Field label="Name" required invalid={submitted && !!errors.label} errorText={errors.label}>
                  <Input value={label} onChange={(e) => setLabel(e.target.value)} placeholder="Issue tracker" autoFocus />
                </Field>
                <Field
                  label="Prefix of its tools"
                  required={creating}
                  invalid={submitted && !!errors.name}
                  errorText={errors.name}
                  helperText={
                    <>
                      Agents name its tools <Code fontSize="xs">{effectiveName || "name"}.tool</Code>. It cannot be changed later.
                    </>
                  }
                >
                  <Input
                    value={effectiveName}
                    disabled={!creating}
                    onChange={(e) => {
                      setNameTouched(true);
                      setName(e.target.value);
                    }}
                  />
                </Field>
                {custom ? (
                  <>
                    <Field
                      label="Server URL"
                      required
                      invalid={submitted && !!errors.url}
                      errorText={errors.url}
                      helperText="Its Streamable HTTP endpoint, often ending in /mcp."
                    >
                      <Input type="url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://mcp.example.com/mcp" />
                    </Field>
                    <Field label="Authentication">
                      <NativeSelect.Root>
                        <NativeSelect.Field value={auth} onChange={(e) => setAuth(e.target.value as ConnectionAuth)}>
                          <option value="bearer">Bearer token (Authorization header)</option>
                          <option value="header">Token in another header</option>
                          <option value="none">None</option>
                        </NativeSelect.Field>
                        <NativeSelect.Indicator />
                      </NativeSelect.Root>
                    </Field>
                    {auth === "header" && (
                      <Field label="Header name" required invalid={submitted && !!errors.headerName} errorText={errors.headerName}>
                        <Input value={headerName} onChange={(e) => setHeaderName(e.target.value)} placeholder="X-API-Key" />
                      </Field>
                    )}
                  </>
                ) : (
                  <Box fontSize="sm" color="gray.600">
                    Server: <Code fontSize="xs">{url}</Code>
                  </Box>
                )}
                {auth !== "none" && (
                  <Field
                    label={tokenLabel}
                    required={creating}
                    invalid={submitted && !!errors.token}
                    errorText={errors.token}
                    helperText={
                      <>
                        {!creating && connection.hasToken ? "A token is stored: leave empty to keep it. " : ""}
                        {preset?.tokenHelp}{" "}
                        {preset?.tokenUrl && (
                          <Link href={preset.tokenUrl} target="_blank" rel="noreferrer" color="purple.600">
                            Get one <FiExternalLink />
                          </Link>
                        )}
                      </>
                    }
                  >
                    <Input
                      type="password"
                      autoComplete="off"
                      value={token}
                      onChange={(e) => setToken(e.target.value)}
                      placeholder={!creating && connection.hasToken ? "••••••••" : ""}
                    />
                  </Field>
                )}
                {auth !== "none" && !encrypted && (
                  <Alert.Root status="warning" borderRadius="md">
                    <Alert.Indicator />
                    <Alert.Content>
                      <Alert.Description fontSize="sm">
                        Tokens are stored unencrypted: set <Code fontSize="xs">jwt_secret</Code> (GOHEAD_JWT_SECRET) to encrypt them.
                      </Alert.Description>
                    </Alert.Content>
                  </Alert.Root>
                )}
              </VStack>
            </DialogBody>
            <DialogFooter gap={3}>
              {creating && (
                <Button variant="plain" me="auto" px={0} onClick={() => setService(null)} disabled={saving}>
                  Back
                </Button>
              )}
              <Button variant="subtle" onClick={onClose} disabled={saving}>
                Cancel
              </Button>
              <Button type="submit" colorPalette="blue" loading={saving}>
                {creating ? "Connect" : "Save"}
              </Button>
            </DialogFooter>
          </form>
        )}
        <DialogCloseTrigger disabled={saving} />
      </DialogContent>
    </DialogRoot>
  );
}

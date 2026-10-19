import { type FormEvent, useState } from "react";
import {
  Alert,
  AlertDescription,
  AlertIcon,
  Button,
  Checkbox,
  CheckboxGroup,
  Code,
  Drawer,
  DrawerBody,
  DrawerCloseButton,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerOverlay,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  Radio,
  RadioGroup,
  Spinner,
  Stack,
  Switch,
  Text,
  VStack,
  Wrap,
  WrapItem,
} from "@chakra-ui/react";
import { api } from "../../../services/api";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import type { Schema } from "../../../shared/types";
import { errorMessage } from "../../../shared/utils/errors";
import { HeadersEditor } from "./HeadersEditor";
import { headerRowsError, headersFromRows, isHttpUrl, newHeaderRow } from "./settingsFormat";
import { type HeaderRow, WEBHOOK_EVENTS, type Webhook, type WebhookFormat, type WebhookInput } from "./types";

function useCollectionNames() {
  return useApiQuery("/admin/collections", async () => {
    const res = await getJSON<{ schema?: Schema }[]>("/admin/collections", "Failed to load collections");
    const list = Array.isArray(res.data) ? res.data : [];
    return list.map((c) => c.schema?.collectionName ?? "").filter(Boolean);
  });
}

/** "*" (all) is shown as nothing selected. */
function selection(list: string[] | undefined): string[] {
  return (list ?? []).filter((x) => x !== "*");
}

/**
 * Creates (webhook = null) or edits a webhook. Mount it with a key per
 * webhook so its fields start from that webhook.
 */
export function WebhookFormDrawer({
  isOpen,
  webhook,
  onClose,
  onSaved,
}: {
  isOpen: boolean;
  webhook: Webhook | null;
  onClose: () => void;
  onSaved: (saved: Webhook, created: boolean) => void;
}) {
  const creating = webhook === null;
  const [name, setName] = useState(webhook?.name ?? "");
  const [url, setUrl] = useState(webhook?.url ?? "");
  const [events, setEvents] = useState<string[]>(selection(webhook?.events));
  const [collections, setCollections] = useState<string[]>(selection(webhook?.collections));
  const [format, setFormat] = useState<WebhookFormat>(webhook?.format === "slack" ? "slack" : "json");
  const [enabled, setEnabled] = useState(webhook?.enabled ?? true);
  const [signed, setSigned] = useState(webhook?.signed ?? true);
  const [rotate, setRotate] = useState(false);
  // On edit, header values are not returned: they are only sent when replaced.
  const [replaceHeaders, setReplaceHeaders] = useState(creating);
  const [rows, setRows] = useState<HeaderRow[]>(() => (webhook?.headers ?? []).map((h) => newHeaderRow(h)));
  const [submitted, setSubmitted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);
  const cols = useCollectionNames();

  // Collections already filtered on but since removed stay selectable.
  const collectionOptions = [...new Set([...(cols.data ?? []), ...collections])];

  const nameError = name.trim() ? null : "Name is required";
  const urlError = !url.trim() ? "URL is required" : !isHttpUrl(url) ? "Must be an absolute http(s) URL" : null;
  const headersError = replaceHeaders ? headerRowsError(rows) : null;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    if (nameError || urlError || headersError) return;
    const body: WebhookInput = { name: name.trim(), url: url.trim(), events, collections, format, enabled, signed };
    if (replaceHeaders) body.headers = headersFromRows(rows);
    if (!creating && signed && webhook.signed && rotate) body.rotate_secret = true;
    setSaving(true);
    setServerError(null);
    try {
      const res = webhook
        ? await api<Webhook>("PUT", `/admin/webhooks/${webhook.id}`, body)
        : await api<Webhook>("POST", "/admin/webhooks", body);
      onSaved(res.data, creating);
    } catch (err) {
      setServerError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Drawer isOpen={isOpen} onClose={onClose} size="md" closeOnOverlayClick={!saving}>
      <DrawerOverlay />
      <DrawerContent as="form" onSubmit={submit} noValidate>
        <DrawerCloseButton isDisabled={saving} />
        <DrawerHeader borderBottomWidth="1px">{creating ? "New webhook" : `Edit ${webhook.name}`}</DrawerHeader>
        <DrawerBody>
          <VStack spacing={5} align="stretch" py={2}>
            {serverError && (
              <Alert status="error" borderRadius="md">
                <AlertIcon />
                <AlertDescription>{serverError}</AlertDescription>
              </Alert>
            )}
            <FormControl isRequired isInvalid={submitted && !!nameError}>
              <FormLabel>Name</FormLabel>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Rebuild website" autoFocus />
              <FormErrorMessage>{nameError}</FormErrorMessage>
            </FormControl>
            <FormControl isRequired isInvalid={submitted && !!urlError}>
              <FormLabel>URL</FormLabel>
              <Input type="url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://example.com/hooks/gohead" />
              <FormErrorMessage>{urlError}</FormErrorMessage>
            </FormControl>

            <FormControl as="fieldset">
              <FormLabel as="legend">Events</FormLabel>
              <CheckboxGroup value={events} onChange={(v) => setEvents(v.map(String))}>
                <Wrap spacingX={4} spacingY={1}>
                  {WEBHOOK_EVENTS.map((ev) => (
                    <WrapItem key={ev}>
                      <Checkbox value={ev}>{ev}</Checkbox>
                    </WrapItem>
                  ))}
                </Wrap>
              </CheckboxGroup>
              <FormHelperText>{events.length === 0 ? "None selected: every event is sent." : `${events.length} selected.`}</FormHelperText>
            </FormControl>

            <FormControl as="fieldset">
              <FormLabel as="legend">Collections</FormLabel>
              {cols.loading && !cols.data ? (
                <Spinner size="sm" />
              ) : cols.error ? (
                <Text color="red.500" fontSize="sm">
                  {cols.error}
                </Text>
              ) : collectionOptions.length === 0 ? (
                <Text color="gray.500" fontSize="sm">
                  No collections yet.
                </Text>
              ) : (
                <CheckboxGroup value={collections} onChange={(v) => setCollections(v.map(String))}>
                  <Wrap spacingX={4} spacingY={1}>
                    {collectionOptions.map((c) => (
                      <WrapItem key={c}>
                        <Checkbox value={c}>{c}</Checkbox>
                      </WrapItem>
                    ))}
                  </Wrap>
                </CheckboxGroup>
              )}
              <FormHelperText>
                {collections.length === 0 ? "None selected: events from every collection are sent." : `Only ${collections.join(", ")}.`}
              </FormHelperText>
            </FormControl>

            <FormControl as="fieldset">
              <FormLabel as="legend">Format</FormLabel>
              <RadioGroup value={format} onChange={(v) => setFormat(v as WebhookFormat)}>
                <Stack direction="row" spacing={6}>
                  <Radio value="json">JSON</Radio>
                  <Radio value="slack">Slack</Radio>
                </Stack>
              </RadioGroup>
              <FormHelperText>
                {format === "json" ? "The full event (item included) as JSON." : 'A {"text": "…"} message for Slack-compatible incoming webhooks.'}
              </FormHelperText>
            </FormControl>

            <FormControl display="flex" alignItems="center" gap={3}>
              <Switch id="webhook-enabled" isChecked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
              <FormLabel htmlFor="webhook-enabled" mb={0}>
                Enabled
              </FormLabel>
            </FormControl>

            <FormControl>
              <Stack spacing={2}>
                <Stack direction="row" align="center" spacing={3}>
                  <Switch id="webhook-signed" isChecked={signed} onChange={(e) => setSigned(e.target.checked)} />
                  <FormLabel htmlFor="webhook-signed" mb={0}>
                    Sign deliveries
                  </FormLabel>
                </Stack>
                <FormHelperText mt={0}>
                  Adds an <Code fontSize="xs">X-Gohead-Signature</Code> header (HMAC-SHA256 of <Code fontSize="xs">timestamp.body</Code>) so the
                  receiver can check the request came from GoHead.{" "}
                  {creating || (signed && !webhook.signed) ? "A secret is generated and shown once after saving." : ""}
                </FormHelperText>
                {!creating && webhook.signed && signed && (
                  <Checkbox isChecked={rotate} onChange={(e) => setRotate(e.target.checked)}>
                    Generate a new secret (the old one stops working)
                  </Checkbox>
                )}
              </Stack>
            </FormControl>

            <FormControl isInvalid={submitted && !!headersError}>
              <FormLabel as="div">Custom headers</FormLabel>
              {replaceHeaders ? (
                <HeadersEditor rows={rows} onChange={setRows} />
              ) : (
                <Stack spacing={2} align="start">
                  <Text fontSize="sm" color="gray.600">
                    {webhook?.headers.length ? `${webhook.headers.join(", ")} (values hidden)` : "None"}
                  </Text>
                  <Button size="sm" variant="outline" onClick={() => setReplaceHeaders(true)}>
                    {webhook?.headers.length ? "Replace headers" : "Add headers"}
                  </Button>
                </Stack>
              )}
              <FormErrorMessage>{headersError}</FormErrorMessage>
              <FormHelperText>
                Sent with every delivery, e.g. a token the receiver expects. Values are stored but never shown again
                {replaceHeaders && !creating ? ": re-enter them all, saving replaces the whole list." : "."}
              </FormHelperText>
            </FormControl>
          </VStack>
        </DrawerBody>
        <DrawerFooter borderTopWidth="1px" gap={3}>
          <Button onClick={onClose} isDisabled={saving}>
            Cancel
          </Button>
          <Button type="submit" colorScheme="blue" isLoading={saving}>
            {creating ? "Create webhook" : "Save changes"}
          </Button>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}

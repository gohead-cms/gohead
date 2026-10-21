import { type FormEvent, useState } from "react";
import {
  Alert,
  Box,
  Button,
  CheckboxGroup,
  Code,
  Field,
  Flex,
  Input,
  Spinner,
  Stack,
  Text,
  VStack,
  Wrap,
  WrapItem,
} from "@chakra-ui/react";
import { Checkbox } from "@/components/ui/checkbox";
import { DrawerBody, DrawerCloseTrigger, DrawerContent, DrawerFooter, DrawerHeader, DrawerRoot, DrawerTitle } from "@/components/ui/drawer";
import { Field as FormField } from "@/components/ui/field";
import { Radio, RadioGroup } from "@/components/ui/radio";
import { Switch } from "@/components/ui/switch";
import { api } from "../../../services/api";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import type { Schema } from "../../../shared/types";
import { errorMessage } from "../../../shared/utils/errors";
import { HeadersEditor } from "./HeadersEditor";
import { headerRowsError, headersFromRows, isHttpUrl, newHeaderRow } from "./settingsFormat";
import { type HeaderRow, WEBHOOK_EVENTS, type Webhook, type WebhookFormat, type WebhookInput } from "./types";

/** Collection names, then single types as "singleton:<name>" (as webhook events name them). */
function useCollectionNames() {
  return useApiQuery("/admin/collections?pageSize=100", async () => {
    const [cols, singles] = await Promise.all([
      getJSON<{ schema?: Schema }[]>("/admin/collections?pageSize=100", "Failed to load collections"),
      getJSON<{ name?: string }[]>("/admin/singletons", "Failed to load single types"),
    ]);
    const names = (Array.isArray(cols.data) ? cols.data : []).map((c) => c.schema?.collectionName ?? "").filter(Boolean);
    const single = (Array.isArray(singles.data) ? singles.data : []).map((s) => (s.name ? `singleton:${s.name}` : "")).filter(Boolean);
    return [...names, ...single];
  });
}

const optionLabel = (key: string) => (key.startsWith("singleton:") ? `${key.slice(10)} (single type)` : key);

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
    const body: WebhookInput = {
      name: name.trim(),
      url: url.trim(),
      events,
      collections,
      format,
      enabled,
      signed,
    };
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
    <DrawerRoot
      open={isOpen}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      size="md"
      closeOnInteractOutside={!saving}
    >
      <DrawerContent>
        <Flex asChild direction="column" h="full">
          <form onSubmit={submit} noValidate>
            <DrawerCloseTrigger disabled={saving} />
            <DrawerHeader borderBottomWidth="1px">
              <DrawerTitle>{creating ? "New webhook" : `Edit ${webhook.name}`}</DrawerTitle>
            </DrawerHeader>
            <DrawerBody>
              <VStack gap={5} align="stretch" py={2}>
                {serverError && (
                  <Alert.Root role="alert" status="error" borderRadius="md">
                    <Alert.Indicator />
                    <Alert.Content>
                      <Alert.Description>{serverError}</Alert.Description>
                    </Alert.Content>
                  </Alert.Root>
                )}
                <FormField label="Name" required invalid={submitted && !!nameError} errorText={nameError}>
                  <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Rebuild website" autoFocus />
                </FormField>
                <FormField label="URL" required invalid={submitted && !!urlError} errorText={urlError}>
                  <Input type="url" value={url} onChange={(e) => setUrl(e.target.value)} placeholder="https://example.com/hooks/gohead" />
                </FormField>

                <Box as="fieldset" minW={0}>
                  <Text as="legend" fontSize="sm" fontWeight="medium" mb={2}>
                    Events
                  </Text>
                  <CheckboxGroup value={events} onValueChange={setEvents}>
                    <Wrap columnGap={4} rowGap={1}>
                      {WEBHOOK_EVENTS.map((ev) => (
                        <WrapItem key={ev}>
                          <Checkbox size="sm" value={ev}>
                            {ev}
                          </Checkbox>
                        </WrapItem>
                      ))}
                    </Wrap>
                  </CheckboxGroup>
                  <Text mt={2} fontSize="sm" color="gray.600">
                    {events.length === 0 ? "None selected: every event is sent." : `${events.length} selected.`}
                  </Text>
                </Box>

                <Box as="fieldset" minW={0}>
                  <Text as="legend" fontSize="sm" fontWeight="medium" mb={2}>
                    Collections and single types
                  </Text>
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
                    <CheckboxGroup value={collections} onValueChange={setCollections}>
                      <Wrap columnGap={4} rowGap={1}>
                        {collectionOptions.map((c) => (
                          <WrapItem key={c}>
                            <Checkbox size="sm" value={c}>
                              {optionLabel(c)}
                            </Checkbox>
                          </WrapItem>
                        ))}
                      </Wrap>
                    </CheckboxGroup>
                  )}
                  <Text mt={2} fontSize="sm" color="gray.600">
                    {collections.length === 0
                      ? "None selected: events from every collection and single type are sent."
                      : `Only ${collections.map(optionLabel).join(", ")}.`}
                  </Text>
                </Box>

                <Box as="fieldset" minW={0}>
                  <Text as="legend" fontSize="sm" fontWeight="medium" mb={2}>
                    Format
                  </Text>
                  <RadioGroup size="sm" value={format} onValueChange={(e) => setFormat(e.value === "slack" ? "slack" : "json")}>
                    <Stack direction="row" gap={6}>
                      <Radio value="json">JSON</Radio>
                      <Radio value="slack">Slack</Radio>
                    </Stack>
                  </RadioGroup>
                  <Text mt={2} fontSize="sm" color="gray.600">
                    {format === "json"
                      ? "The full event (item included) as JSON."
                      : 'A {"text": "…"} message for Slack-compatible incoming webhooks.'}
                  </Text>
                </Box>

                <Switch size="sm" checked={enabled} onCheckedChange={(e) => setEnabled(e.checked)}>
                  Enabled
                </Switch>

                <Field.Root>
                  <Stack gap={2} w="full">
                    <Switch size="sm" checked={signed} onCheckedChange={(e) => setSigned(e.checked)}>
                      Sign deliveries
                    </Switch>
                    <Field.HelperText mt={0}>
                      Adds an <Code fontSize="xs">X-Gohead-Signature</Code> header (HMAC-SHA256 of <Code fontSize="xs">timestamp.body</Code>
                      ) so the receiver can check the request came from GoHead.{" "}
                      {creating || (signed && !webhook.signed) ? "A secret is generated and shown once after saving." : ""}
                    </Field.HelperText>
                    {!creating && webhook.signed && signed && (
                      <Checkbox size="sm" checked={rotate} onCheckedChange={(e) => setRotate(e.checked === true)}>
                        Generate a new secret (the old one stops working)
                      </Checkbox>
                    )}
                  </Stack>
                </Field.Root>

                <Field.Root invalid={submitted && !!headersError}>
                  <Text fontWeight="medium" fontSize="sm">
                    Custom headers
                  </Text>
                  <Field.ErrorText mt={0} mb={2}>
                    {headersError}
                  </Field.ErrorText>
                  {replaceHeaders ? (
                    <HeadersEditor rows={rows} onChange={setRows} />
                  ) : (
                    <Stack gap={2} align="start">
                      <Text fontSize="sm" color="gray.600">
                        {webhook?.headers.length ? `${webhook.headers.join(", ")} (values hidden)` : "None"}
                      </Text>
                      <Button size="xs" variant="outline" onClick={() => setReplaceHeaders(true)}>
                        {webhook?.headers.length ? "Replace headers" : "Add headers"}
                      </Button>
                    </Stack>
                  )}
                  <Field.HelperText>
                    Sent with every delivery, e.g. a token the receiver expects. Values are stored but never shown again
                    {replaceHeaders && !creating ? ": re-enter them all, saving replaces the whole list." : "."}
                  </Field.HelperText>
                </Field.Root>
              </VStack>
            </DrawerBody>
            <DrawerFooter borderTopWidth="1px" gap={3}>
              <Button variant="subtle" onClick={onClose} disabled={saving}>
                Cancel
              </Button>
              <Button type="submit" colorPalette="blue" loading={saving}>
                {creating ? "Create webhook" : "Save changes"}
              </Button>
            </DrawerFooter>
          </form>
        </Flex>
      </DrawerContent>
    </DrawerRoot>
  );
}

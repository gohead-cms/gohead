import { type FormEvent, useState } from "react";
import { Alert, Button, Input, NativeSelect, NumberInput, VStack } from "@chakra-ui/react";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { api } from "../../../services/api";
import { errorMessage } from "../../../shared/utils/errors";
import { RoleSelect } from "./RoleSelect";
import type { CreatedApiToken } from "./types";

const MAX_DAYS = 3650;
const EXPIRY_PRESETS = [
  { value: "30", label: "30 days" },
  { value: "90", label: "90 days" },
  { value: "365", label: "1 year" },
  { value: "never", label: "Never" },
  { value: "custom", label: "Custom…" },
];

/** Creates an API token; onCreated receives it with its one-time secret. */
export function TokenFormModal({
  isOpen,
  onClose,
  onCreated,
}: {
  isOpen: boolean;
  onClose: () => void;
  onCreated: (t: CreatedApiToken) => void;
}) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [role, setRole] = useState("viewer");
  const [expiry, setExpiry] = useState("90");
  const [customDays, setCustomDays] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const nameError = name.trim() ? null : "Give the token a name, e.g. the app that uses it";
  const days = expiry === "custom" ? Number(customDays) : expiry === "never" ? null : Number(expiry);
  const daysError =
    expiry === "custom" && (!Number.isInteger(days) || days === null || days < 1 || days > MAX_DAYS)
      ? `A whole number of days between 1 and ${MAX_DAYS}`
      : null;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    if (nameError || daysError) return;
    setSaving(true);
    setServerError(null);
    try {
      const body: Record<string, unknown> = {
        name: name.trim(),
        description: description.trim(),
        role,
      };
      if (days !== null) body.expires_in_days = days;
      const res = await api<CreatedApiToken>("POST", "/admin/tokens", body);
      onCreated(res.data);
    } catch (err) {
      setServerError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <DialogRoot
      open={isOpen}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      closeOnInteractOutside={!saving}
    >
      <DialogContent>
        <form onSubmit={submit} noValidate>
          <DialogHeader>
            <DialogTitle>New API token</DialogTitle>
          </DialogHeader>
          <DialogCloseTrigger disabled={saving} />
          <DialogBody>
            <VStack gap={4} align="stretch">
              {serverError && (
                <Alert.Root status="error" borderRadius="md">
                  <Alert.Indicator />
                  <Alert.Content>
                    <Alert.Description>{serverError}</Alert.Description>
                  </Alert.Content>
                </Alert.Root>
              )}
              <Field label="Name" required invalid={submitted && !!nameError} errorText={nameError}>
                <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="website, ci-deploy…" autoFocus />
              </Field>
              <Field label="Description">
                <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="What uses it, who to ask" />
              </Field>
              <RoleSelect value={role} onChange={setRole} />
              <Field label="Expires" helperText="Short-lived tokens limit the damage if one leaks.">
                <NativeSelect.Root>
                  <NativeSelect.Field value={expiry} onChange={(e) => setExpiry(e.target.value)}>
                    {EXPIRY_PRESETS.map((p) => (
                      <option key={p.value} value={p.value}>
                        {p.label}
                      </option>
                    ))}
                  </NativeSelect.Field>
                  <NativeSelect.Indicator />
                </NativeSelect.Root>
              </Field>
              {expiry === "custom" && (
                <Field label="Days until expiry" required invalid={submitted && !!daysError} errorText={daysError}>
                  <NumberInput.Root min={1} max={MAX_DAYS} value={customDays} onValueChange={(e) => setCustomDays(e.value)} w="full">
                    <NumberInput.Input />
                  </NumberInput.Root>
                </Field>
              )}
            </VStack>
          </DialogBody>
          <DialogFooter gap={3}>
            <Button variant="subtle" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button type="submit" colorPalette="blue" loading={saving}>
              Create token
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </DialogRoot>
  );
}

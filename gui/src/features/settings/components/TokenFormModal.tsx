import { type FormEvent, useState } from "react";
import {
  Alert,
  AlertDescription,
  AlertIcon,
  Button,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  NumberInput,
  NumberInputField,
  Select,
  VStack,
} from "@chakra-ui/react";
import { api } from "../../../services/api";
import { errorMessage } from "../../../shared/utils/errors";
import { RoleSelect } from "./RoleSelect";
import type { CreatedApiToken, RoleName } from "./types";

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
  const [role, setRole] = useState<RoleName>("viewer");
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
      const body: Record<string, unknown> = { name: name.trim(), description: description.trim(), role };
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
    <Modal isOpen={isOpen} onClose={onClose} closeOnOverlayClick={!saving}>
      <ModalOverlay />
      <ModalContent as="form" onSubmit={submit} noValidate>
        <ModalHeader>New API token</ModalHeader>
        <ModalCloseButton isDisabled={saving} />
        <ModalBody>
          <VStack spacing={4} align="stretch">
            {serverError && (
              <Alert status="error" borderRadius="md">
                <AlertIcon />
                <AlertDescription>{serverError}</AlertDescription>
              </Alert>
            )}
            <FormControl isRequired isInvalid={submitted && !!nameError}>
              <FormLabel>Name</FormLabel>
              <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="website, ci-deploy…" autoFocus />
              <FormErrorMessage>{nameError}</FormErrorMessage>
            </FormControl>
            <FormControl>
              <FormLabel>Description</FormLabel>
              <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder="What uses it, who to ask" />
            </FormControl>
            <RoleSelect value={role} onChange={setRole} />
            <FormControl>
              <FormLabel>Expires</FormLabel>
              <Select value={expiry} onChange={(e) => setExpiry(e.target.value)}>
                {EXPIRY_PRESETS.map((p) => (
                  <option key={p.value} value={p.value}>
                    {p.label}
                  </option>
                ))}
              </Select>
              <FormHelperText>Short-lived tokens limit the damage if one leaks.</FormHelperText>
            </FormControl>
            {expiry === "custom" && (
              <FormControl isRequired isInvalid={submitted && !!daysError}>
                <FormLabel>Days until expiry</FormLabel>
                <NumberInput min={1} max={MAX_DAYS} value={customDays} onChange={(v) => setCustomDays(v)}>
                  <NumberInputField />
                </NumberInput>
                <FormErrorMessage>{daysError}</FormErrorMessage>
              </FormControl>
            )}
          </VStack>
        </ModalBody>
        <ModalFooter gap={3}>
          <Button onClick={onClose} isDisabled={saving}>
            Cancel
          </Button>
          <Button type="submit" colorScheme="blue" isLoading={saving}>
            Create token
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}

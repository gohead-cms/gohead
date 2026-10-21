import { type FormEvent, useState } from "react";
import { Alert, Button, Code, Input, VStack } from "@chakra-ui/react";
import { DialogBody, DialogCloseTrigger, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { Field } from "@/components/ui/field";
import { api } from "../../../services/api";
import type { Locale } from "../../../shared/hooks/useLocales";
import { errorMessage } from "../../../shared/utils/errors";

/** Codes offered in the code field's suggestions, with a default name. */
const COMMON_LOCALES: { code: string; name: string }[] = [
  { code: "en", name: "English" },
  { code: "fr", name: "Français" },
  { code: "de", name: "Deutsch" },
  { code: "es", name: "Español" },
  { code: "it", name: "Italiano" },
  { code: "pt", name: "Português" },
  { code: "pt-BR", name: "Português (Brasil)" },
  { code: "nl", name: "Nederlands" },
  { code: "pl", name: "Polski" },
  { code: "ja", name: "日本語" },
  { code: "zh-Hans", name: "简体中文" },
];

// Same rule as the API: a BCP 47 style tag such as "fr", "pt-BR", "zh-Hant".
const LOCALE_CODE = /^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$/;

/** Adds a locale (locale null) or renames one. */
export function LocaleFormModal({
  isOpen,
  locale,
  existingCodes,
  onClose,
  onSaved,
}: {
  isOpen: boolean;
  /** The locale to rename, or null to add one. */
  locale: Locale | null;
  existingCodes: string[];
  onClose: () => void;
  onSaved: (l: Locale) => void;
}) {
  const editing = !!locale;
  const [code, setCode] = useState(locale?.code ?? "");
  const [name, setName] = useState(locale?.name ?? "");
  const [submitted, setSubmitted] = useState(false);
  const [saving, setSaving] = useState(false);
  const [serverError, setServerError] = useState<string | null>(null);

  const trimmed = code.trim();
  const codeError = editing
    ? null
    : !trimmed
      ? "Enter a locale code, e.g. fr"
      : !LOCALE_CODE.test(trimmed)
        ? "A lower-case language code, optionally followed by a region or script: fr, pt-BR, zh-Hans"
        : existingCodes.includes(trimmed)
          ? "This locale already exists"
          : null;
  const nameError = name.trim() ? null : "Give the locale a name, e.g. Français";

  const onCodeChange = (value: string) => {
    setCode(value);
    // Picking a suggested code fills in its name, unless one was typed.
    const known = COMMON_LOCALES.find((l) => l.code === value.trim());
    if (known && !name.trim()) setName(known.name);
  };

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    if (codeError || nameError) return;
    setSaving(true);
    setServerError(null);
    try {
      const res = editing
        ? await api<Locale>("PUT", `/admin/locales/${encodeURIComponent(locale.code)}`, { name: name.trim() })
        : await api<Locale>("POST", "/admin/locales", {
            code: trimmed,
            name: name.trim(),
          });
      onSaved(res.data);
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
            <DialogTitle>{editing ? `Rename ${locale.code}` : "Add a locale"}</DialogTitle>
          </DialogHeader>
          <DialogCloseTrigger disabled={saving} />
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
              <Field
                label="Code"
                required={!editing}
                invalid={submitted && !!codeError}
                errorText={codeError}
                helperText={
                  submitted && codeError ? undefined : editing ? (
                    "The code cannot change: API clients ask for content with it."
                  ) : (
                    <>
                      A BCP 47 language tag: a language (<Code fontSize="xs">fr</Code>), optionally with a region (
                      <Code fontSize="xs">pt-BR</Code>) or script (<Code fontSize="xs">zh-Hans</Code>). Clients read content with{" "}
                      <Code fontSize="xs">?locale=</Code> and this code.
                    </>
                  )
                }
              >
                <Input
                  value={code}
                  onChange={(e) => onCodeChange(e.target.value)}
                  placeholder="fr"
                  list={editing ? undefined : "common-locale-codes"}
                  readOnly={editing}
                  bg={editing ? "gray.100" : undefined}
                  fontFamily="mono"
                  autoFocus={!editing}
                  autoComplete="off"
                />
                {!editing && (
                  <datalist id="common-locale-codes">
                    {COMMON_LOCALES.filter((l) => !existingCodes.includes(l.code)).map((l) => (
                      <option key={l.code} value={l.code}>
                        {l.name}
                      </option>
                    ))}
                  </datalist>
                )}
              </Field>
              <Field label="Name" required invalid={submitted && !!nameError} errorText={nameError}>
                <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Français" autoFocus={editing} />
              </Field>
            </VStack>
          </DialogBody>
          <DialogFooter gap={3}>
            <Button variant="subtle" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button type="submit" colorPalette="blue" loading={saving}>
              {editing ? "Save" : "Add locale"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </DialogRoot>
  );
}

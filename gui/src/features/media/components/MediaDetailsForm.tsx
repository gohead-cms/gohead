import { type FormEvent, useState } from "react";
import { Alert, Button, Field, Input, Stack, Textarea } from "@chakra-ui/react";
import { api } from "../../../services/api";
import { useNotify } from "../../../shared/hooks/useNotify";
import { errorMessage } from "../../../shared/utils/errors";
import type { MediaItem } from "../media";

/** Edits a file's name, alt text and caption (PUT /api/media/{id}). */
export function MediaDetailsForm({
  item,
  canWrite,
  onSaved,
}: {
  item: MediaItem;
  canWrite: boolean;
  onSaved: (item: MediaItem) => void;
}) {
  const [name, setName] = useState(item.name);
  const [alt, setAlt] = useState(item.alt);
  const [caption, setCaption] = useState(item.caption);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const notify = useNotify();

  const nameError = name.trim() === "" ? "The name cannot be empty" : null;
  const dirty = name !== item.name || alt !== item.alt || caption !== item.caption;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (nameError || !dirty) return;
    setSaving(true);
    setError(null);
    try {
      const res = await api<MediaItem>("PUT", `/api/media/${item.id}`, { data: { name: name.trim(), alt, caption } });
      notify.success("File details saved");
      onSaved(res.data);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={submit} noValidate>
      <Stack gap={3}>
        {error && (
          <Alert.Root role="alert" status="error" borderRadius="md" fontSize="sm">
            <Alert.Indicator />
            <Alert.Content>{error}</Alert.Content>
          </Alert.Root>
        )}
        <Field.Root invalid={!!nameError} readOnly={!canWrite} required>
          <Field.Label fontSize="sm">
            Name
            <Field.RequiredIndicator />
          </Field.Label>
          <Input size="xs" value={name} onChange={(e) => setName(e.target.value)} maxLength={255} />
          {nameError && <Field.ErrorText>{nameError}</Field.ErrorText>}
        </Field.Root>
        <Field.Root readOnly={!canWrite}>
          <Field.Label fontSize="sm">Alt text</Field.Label>
          <Input size="xs" value={alt} onChange={(e) => setAlt(e.target.value)} placeholder="Describe the image for screen readers" />
        </Field.Root>
        <Field.Root readOnly={!canWrite}>
          <Field.Label fontSize="sm">Caption</Field.Label>
          <Textarea size="sm" rows={3} value={caption} onChange={(e) => setCaption(e.target.value)} />
        </Field.Root>
        {canWrite && (
          <Button type="submit" size="xs" colorPalette="blue" alignSelf="flex-start" loading={saving} disabled={!dirty || !!nameError}>
            Save details
          </Button>
        )}
      </Stack>
    </form>
  );
}

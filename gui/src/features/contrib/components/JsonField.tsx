import { useState } from "react";
import { Text, Textarea, VStack } from "@chakra-ui/react";
import { InvalidValue } from "./entryValues";

interface JsonFieldProps {
  id: string;
  value: unknown;
  onChange: (value: unknown) => void;
}

const show = (value: unknown) =>
  value === undefined || value === null || value === "" || value instanceof InvalidValue ? "" : JSON.stringify(value, null, 2);

/**
 * Edits a json attribute as text. The form gets the parsed value, nothing
 * when the text is empty (or null), or an InvalidValue (which blocks saving)
 * while the text does not parse. The text is kept as typed.
 */
export function JsonField({ id, value, onChange }: JsonFieldProps) {
  const [text, setText] = useState(() => show(value));
  const [error, setError] = useState<string | null>(null);
  // The value this field last sent (or started from): another value comes
  // from outside (a translation's "Use default") and replaces the text.
  const [own, setOwn] = useState<unknown>(value);
  if (value !== own) {
    setOwn(value);
    setText(show(value));
    setError(null);
  }

  const send = (v: unknown) => {
    setOwn(v);
    onChange(v);
  };
  const edit = (next: string) => {
    setText(next);
    if (next.trim() === "") {
      setError(null);
      send(undefined);
      return;
    }
    try {
      const parsed: unknown = JSON.parse(next);
      setError(null);
      send(parsed);
    } catch (err) {
      const message = err instanceof Error ? err.message : "invalid JSON";
      setError(message);
      send(new InvalidValue(`not valid JSON (${message})`));
    }
  };

  return (
    <VStack align="stretch" gap={1}>
      <Textarea
        id={id}
        value={text}
        onChange={(e) => edit(e.target.value)}
        rows={Math.min(16, Math.max(4, text.split("\n").length + 1))}
        fontFamily="mono"
        fontSize="sm"
        spellCheck={false}
        placeholder='{"key": "value"}'
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${id}-error` : undefined}
      />
      {error && (
        <Text id={`${id}-error`} fontSize="sm" color="red.500">
          Not valid JSON: {error}
        </Text>
      )}
    </VStack>
  );
}

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
 * when the text is empty, or an InvalidValue (which blocks saving) while the
 * text does not parse.
 */
export function JsonField({ id, value, onChange }: JsonFieldProps) {
  const [text, setText] = useState(() => show(value));
  const [error, setError] = useState<string | null>(null);

  const edit = (next: string) => {
    setText(next);
    if (next.trim() === "") {
      setError(null);
      onChange(undefined);
      return;
    }
    try {
      onChange(JSON.parse(next));
      setError(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : "invalid JSON";
      setError(message);
      onChange(new InvalidValue(`not valid JSON (${message})`));
    }
  };

  return (
    <VStack align="stretch" spacing={1}>
      <Textarea
        id={id}
        value={text}
        onChange={(e) => edit(e.target.value)}
        onBlur={() => {
          // Tidy valid JSON once the field is left.
          if (!error && text.trim() !== "") setText(show(JSON.parse(text)));
        }}
        rows={Math.min(16, Math.max(4, text.split("\n").length + 1))}
        fontFamily="mono"
        fontSize="sm"
        spellCheck={false}
        placeholder='{"key": "value"}'
        isInvalid={!!error}
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

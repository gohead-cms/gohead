import { type ReactNode } from "react";
import { Alert, Box, Button, Field as ChakraField, Heading, Text, useClipboard } from "@chakra-ui/react";
import { FiCheck, FiCopy } from "react-icons/fi";

/** A white card with a title, for one part of a form. */
export function EditorSection({ title, description, actions, children }: { title: string; description?: ReactNode; actions?: ReactNode; children: ReactNode }) {
  return (
    <Box as="section" bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 6 }}>
      <Box display="flex" justifyContent="space-between" alignItems="start" gap={4} mb={4}>
        <Box>
          <Heading size="md" fontWeight="bold">
            {title}
          </Heading>
          {description && (
            <Text fontSize="sm" color="gray.500" mt={1}>
              {description}
            </Text>
          )}
        </Box>
        {actions}
      </Box>
      {children}
    </Box>
  );
}

/** A v2-style alert: an icon and a plain message (no bold title). */
export function Notice({ children, ...rest }: Alert.RootProps) {
  return (
    <Alert.Root fontSize="md" {...rest}>
      <Alert.Indicator />
      <Alert.Content>
        <Alert.Description>{children}</Alert.Description>
      </Alert.Content>
    </Alert.Root>
  );
}

/** A required mark for labels that are not a Field.Label (a fieldset legend). */
export function RequiredMark() {
  return (
    <Text as="span" color="fg.error" ms={1} aria-hidden="true">
      *
    </Text>
  );
}

/** A labelled form control with optional help text and error. */
export function Field({
  label,
  error,
  help,
  isRequired,
  children,
}: {
  label: string;
  error?: string;
  help?: ReactNode;
  isRequired?: boolean;
  children: ReactNode;
}) {
  return (
    <ChakraField.Root invalid={Boolean(error)} required={isRequired} gap={1}>
      <ChakraField.Label fontSize="sm">
        {label}
        <ChakraField.RequiredIndicator />
      </ChakraField.Label>
      {children}
      {error ? (
        <ChakraField.ErrorText fontSize="sm">{error}</ChakraField.ErrorText>
      ) : help ? (
        <ChakraField.HelperText fontSize="sm">{help}</ChakraField.HelperText>
      ) : null}
    </ChakraField.Root>
  );
}

/** Copies a value to the clipboard. */
export function CopyButton({ value, label = "Copy" }: { value: string; label?: string }) {
  const clip = useClipboard({ value });
  return (
    <Button size="xs" variant="outline" onClick={() => clip.copy()} flexShrink={0}>
      {clip.copied ? <FiCheck /> : <FiCopy />}
      {clip.copied ? "Copied" : label}
    </Button>
  );
}

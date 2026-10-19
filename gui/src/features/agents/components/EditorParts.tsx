import { ReactNode } from "react";
import {
  Box,
  Button,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Heading,
  Text,
  useClipboard,
} from "@chakra-ui/react";
import { FiCheck, FiCopy } from "react-icons/fi";

/** A white card with a title, for one part of a form. */
export function EditorSection({ title, description, actions, children }: { title: string; description?: ReactNode; actions?: ReactNode; children: ReactNode }) {
  return (
    <Box as="section" bg="white" borderWidth="1px" borderRadius="lg" p={{ base: 4, md: 6 }}>
      <Box display="flex" justifyContent="space-between" alignItems="start" gap={4} mb={4}>
        <Box>
          <Heading size="sm">{title}</Heading>
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
    <FormControl isInvalid={Boolean(error)} isRequired={isRequired}>
      <FormLabel fontSize="sm" mb={1}>
        {label}
      </FormLabel>
      {children}
      {error ? <FormErrorMessage>{error}</FormErrorMessage> : help ? <FormHelperText>{help}</FormHelperText> : null}
    </FormControl>
  );
}

/** Copies a value to the clipboard. */
export function CopyButton({ value, label = "Copy" }: { value: string; label?: string }) {
  const { onCopy, hasCopied } = useClipboard(value);
  return (
    <Button size="sm" variant="outline" leftIcon={hasCopied ? <FiCheck /> : <FiCopy />} onClick={onCopy} flexShrink={0}>
      {hasCopied ? "Copied" : label}
    </Button>
  );
}

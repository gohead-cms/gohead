import type { ReactNode } from "react";
import { Alert, Button, Code, HStack, Text, VStack, useClipboard } from "@chakra-ui/react";
import { DialogBody, DialogContent, DialogFooter, DialogHeader, DialogRoot, DialogTitle } from "@/components/ui/dialog";
import { FiCheck, FiCopy } from "react-icons/fi";

/**
 * Shows a secret that the server returns only once, with a copy button.
 * It closes only through its button, so it is not dismissed by accident.
 */
export function SecretDialog({
  title,
  secret,
  label,
  children,
  onClose,
}: {
  title: string;
  secret: string | null;
  label: string;
  children?: ReactNode;
  onClose: () => void;
}) {
  const clip = useClipboard({ value: secret ?? "" });
  return (
    <DialogRoot
      open={secret !== null}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      closeOnInteractOutside={false}
      closeOnEscape={false}
      size="xl"
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <VStack align="stretch" gap={4}>
            <Alert.Root status="warning" borderRadius="md">
              <Alert.Indicator />
              <Alert.Content>
                <Alert.Description>Copy it now and store it somewhere safe: it will not be shown again.</Alert.Description>
              </Alert.Content>
            </Alert.Root>
            <VStack align="stretch" gap={1}>
              <Text fontSize="sm" fontWeight="medium">
                {label}
              </Text>
              <HStack align="stretch">
                <Code
                  aria-label={label}
                  flex="1"
                  p={2}
                  fontSize="sm"
                  wordBreak="break-all"
                  whiteSpace="pre-wrap"
                  borderRadius="md"
                  data-testid="secret-value"
                >
                  {secret}
                </Code>
                <Button variant="subtle" onClick={clip.copy} flexShrink={0}>
                  {clip.copied ? <FiCheck /> : <FiCopy />}
                  {clip.copied ? "Copied" : "Copy"}
                </Button>
              </HStack>
            </VStack>
            {children}
          </VStack>
        </DialogBody>
        <DialogFooter>
          <Button colorPalette="blue" onClick={onClose}>
            I have stored it
          </Button>
        </DialogFooter>
      </DialogContent>
    </DialogRoot>
  );
}

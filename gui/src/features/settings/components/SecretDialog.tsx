import type { ReactNode } from "react";
import {
  Alert,
  AlertDescription,
  AlertIcon,
  Button,
  Code,
  HStack,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Text,
  VStack,
  useClipboard,
} from "@chakra-ui/react";
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
  const { hasCopied, onCopy } = useClipboard(secret ?? "");
  return (
    <Modal isOpen={secret !== null} onClose={onClose} closeOnOverlayClick={false} closeOnEsc={false} size="xl">
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>{title}</ModalHeader>
        <ModalBody>
          <VStack align="stretch" spacing={4}>
            <Alert status="warning" borderRadius="md">
              <AlertIcon />
              <AlertDescription>Copy it now and store it somewhere safe: it will not be shown again.</AlertDescription>
            </Alert>
            <VStack align="stretch" spacing={1}>
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
                <Button leftIcon={hasCopied ? <FiCheck /> : <FiCopy />} onClick={onCopy} flexShrink={0}>
                  {hasCopied ? "Copied" : "Copy"}
                </Button>
              </HStack>
            </VStack>
            {children}
          </VStack>
        </ModalBody>
        <ModalFooter>
          <Button colorScheme="blue" onClick={onClose}>
            I have stored it
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  );
}

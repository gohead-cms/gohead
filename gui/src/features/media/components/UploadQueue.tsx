import { Badge, Box, Button, Flex, HStack, IconButton, Spinner, Stack, Text } from "@chakra-ui/react";
import { FiX } from "react-icons/fi";
import { type UploadStatus, type UploadTask, formatBytes } from "../media";

const statusLabel: Record<UploadStatus, { label: string; color: string }> = {
  pending: { label: "Waiting", color: "gray" },
  uploading: { label: "Uploading", color: "blue" },
  done: { label: "Uploaded", color: "green" },
  error: { label: "Failed", color: "red" },
};

/** The files of the current uploads, each with its status or server error. */
export function UploadQueue({
  tasks,
  busy,
  onDismiss,
  onClear,
}: {
  tasks: UploadTask[];
  busy: boolean;
  onDismiss: (id: string) => void;
  onClear: () => void;
}) {
  if (tasks.length === 0) return null;
  const done = tasks.filter((t) => t.status === "done").length;
  const failed = tasks.filter((t) => t.status === "error").length;
  return (
    <Box bg="white" borderWidth="1px" borderRadius="lg" p={4} mb={4} aria-label="Uploads" role="region">
      <Flex justify="space-between" align="center" mb={2} gap={2} wrap="wrap">
        <HStack gap={2}>
          {busy && <Spinner size="sm" color="blue.500" />}
          <Text fontWeight="semibold" fontSize="sm">
            {busy ? "Uploading…" : "Uploads"} {done}/{tasks.length} done
            {failed > 0 && `, ${failed} failed`}
          </Text>
        </HStack>
        {!busy && (
          <Button size="2xs" variant="ghost" onClick={onClear}>
            Clear
          </Button>
        )}
      </Flex>
      <Stack gap={1}>
        {tasks.map((t) => (
          <Flex key={t.id} align="center" gap={3} fontSize="sm" minW={0}>
            <Badge colorPalette={statusLabel[t.status].color} flexShrink={0}>
              {statusLabel[t.status].label}
            </Badge>
            <Box flex="1" minW={0}>
              <Text lineClamp={1} title={t.name}>
                {t.name} <Text as="span" color="gray.500">({formatBytes(t.size)})</Text>
              </Text>
              {t.error && (
                <Text color="red.600" fontSize="xs">
                  {t.error}
                </Text>
              )}
            </Box>
            {(t.status === "done" || t.status === "error") && (
              <IconButton aria-label={`Dismiss ${t.name}`} size="2xs" variant="ghost" onClick={() => onDismiss(t.id)}>
                <FiX />
              </IconButton>
            )}
          </Flex>
        ))}
      </Stack>
    </Box>
  );
}

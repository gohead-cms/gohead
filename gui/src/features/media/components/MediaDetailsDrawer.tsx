import { type ReactNode, useState } from "react";
import {
  Alert,
  Box,
  Button,
  Center,
  chakra,
  Code,
  Field,
  Grid,
  HStack,
  IconButton,
  Image,
  Input,
  Separator,
  Spinner,
  Text,
  useClipboard,
} from "@chakra-ui/react";
import {
  DrawerBody,
  DrawerCloseTrigger,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerRoot,
  DrawerTitle,
} from "@/components/ui/drawer";
import { Tooltip } from "@/components/ui/tooltip";
import { FiCheck, FiCopy, FiExternalLink, FiTrash2 } from "react-icons/fi";
import { api } from "../../../services/api";
import { getJSON, useApiQuery } from "../../../shared/hooks/useApiQuery";
import { useNotify } from "../../../shared/hooks/useNotify";
import { ConfirmButton } from "../../../shared/ui/ConfirmButton";
import { errorMessage } from "../../../shared/utils/errors";
import { formatTime } from "../../agents/components/runFormat";
import { type MediaItem, absoluteUrl, dimensions, formatBytes, isImage } from "../media";
import { MediaDetailsForm } from "./MediaDetailsForm";
import { MediaThumb } from "./MediaThumb";

function Preview({ item }: { item: MediaItem }) {
  const src = absoluteUrl(item.url);
  if (isImage(item.mime)) {
    return (
      <Center bg="gray.100" borderRadius="md" overflow="hidden">
        <Image src={src} alt={item.alt || item.name} maxH="320px" objectFit="contain" />
      </Center>
    );
  }
  if (item.mime.startsWith("video/")) {
    return <chakra.video src={src} controls preload="metadata" w="100%" maxH="320px" borderRadius="md" bg="black" />;
  }
  if (item.mime.startsWith("audio/")) {
    return (
      <Box>
        <MediaThumb item={item} height="120px" />
        <chakra.audio src={src} controls w="100%" mt={2} />
      </Box>
    );
  }
  return (
    <Box borderRadius="md" overflow="hidden" borderWidth="1px">
      <MediaThumb item={item} height="160px" />
    </Box>
  );
}

function UrlField({ url }: { url: string }) {
  const clip = useClipboard({ value: url });
  return (
    <Field.Root>
      <Field.Label fontSize="sm">URL</Field.Label>
      <HStack gap={1} w="full">
        <Input size="xs" value={url} readOnly fontFamily="mono" fontSize="xs" onFocus={(e) => e.target.select()} />
        <Tooltip content={clip.copied ? "Copied" : "Copy URL"}>
          <IconButton aria-label="Copy URL" size="xs" variant="subtle" onClick={() => clip.copy()}>
            {clip.copied ? <FiCheck /> : <FiCopy />}
          </IconButton>
        </Tooltip>
        <IconButton asChild aria-label="Open file in a new tab" size="xs" variant="subtle">
          <a href={url} target="_blank" rel="noopener noreferrer">
            <FiExternalLink />
          </a>
        </IconButton>
      </HStack>
    </Field.Root>
  );
}

function Facts({ item }: { item: MediaItem }) {
  const rows: [string, ReactNode][] = [
    ["Type", item.mime],
    ["Size", `${formatBytes(item.size)} (${item.size.toLocaleString()} bytes)`],
  ];
  const dims = dimensions(item);
  if (dims) rows.push(["Dimensions", `${dims} px`]);
  rows.push(["Uploaded", formatTime(item.createdAt)]);
  if (item.uploadedBy) rows.push(["By", item.uploadedBy]);
  if (item.hash) {
    rows.push([
      "SHA-256",
      <Code key="hash" fontSize="xs" wordBreak="break-all" bg="transparent" p={0}>
        {item.hash}
      </Code>,
    ]);
  }
  return (
    <Grid templateColumns="96px 1fr" columnGap={3} rowGap={1} fontSize="sm">
      {rows.map(([label, value]) => (
        <Box key={label} display="contents">
          <Text color="gray.500">{label}</Text>
          <Box minW={0} wordBreak="break-word">
            {value}
          </Box>
        </Box>
      ))}
    </Grid>
  );
}

/** One media file: preview, URL, facts, editable details and delete. */
export function MediaDetailsDrawer({
  id,
  canUpdate,
  canDelete,
  onClose,
  onChanged,
}: {
  id: number | null;
  /** Media update: the details can be edited. */
  canUpdate: boolean;
  /** Media delete: the file can be deleted. */
  canDelete: boolean;
  onClose: () => void;
  onChanged: () => void;
}) {
  const query = useApiQuery<MediaItem>(id === null ? null : `/api/media/${id}`, async () =>
    (await getJSON<MediaItem>(`/api/media/${id}`, "Could not load the file")).data,
  );
  const [deleteError, setDeleteError] = useState<{ id: number; message: string } | null>(null);
  const notify = useNotify();
  const item = query.data;

  const remove = async () => {
    if (!item) return;
    try {
      await api("DELETE", `/api/media/${item.id}`);
      notify.success(`Deleted ${item.name}`);
      setDeleteError(null);
      onChanged();
      onClose();
    } catch (err) {
      setDeleteError({ id: item.id, message: errorMessage(err) });
      notify.failure(err, "Could not delete the file");
    }
  };

  return (
    <DrawerRoot
      open={id !== null}
      onOpenChange={(e) => {
        if (!e.open) onClose();
      }}
      size="md"
      placement="end"
    >
      <DrawerContent>
        <DrawerHeader pr={12}>
          <DrawerTitle fontSize="xl" fontWeight="semibold" lineClamp={1} title={item?.name}>
            {item?.name ?? "File details"}
          </DrawerTitle>
        </DrawerHeader>
        <DrawerBody>
          {query.error && (
            <Alert.Root role="alert" status="error" borderRadius="md">
              <Alert.Indicator />
              <Alert.Content>{query.error}</Alert.Content>
            </Alert.Root>
          )}
          {!item && !query.error && (
            <Center py={12}>
              <Spinner />
            </Center>
          )}
          {item && (
            <Box display="flex" flexDirection="column" gap={5} pb={4}>
              <Preview item={item} />
              <UrlField url={absoluteUrl(item.url)} />
              <Facts item={item} />
              <Separator />
              <MediaDetailsForm
                key={`${item.id}-${item.updatedAt}`}
                item={item}
                canWrite={canUpdate}
                onSaved={() => {
                  query.refetch();
                  onChanged();
                }}
              />
              {deleteError?.id === item.id && (
                <Alert.Root role="alert" status="error" borderRadius="md" fontSize="sm">
                  <Alert.Indicator />
                  <Alert.Content>{deleteError.message}</Alert.Content>
                </Alert.Root>
              )}
            </Box>
          )}
        </DrawerBody>
        {item && canDelete && (
          <DrawerFooter borderTopWidth="1px" justifyContent="space-between">
            <ConfirmButton
              title="Delete this file?"
              body={
                <>
                  <Text mb={2}>
                    <b>{item.name}</b> will be removed from storage. This cannot be undone.
                  </Text>
                  <Text color="gray.600" fontSize="sm">
                    Content that uses it will show an empty media field.
                  </Text>
                </>
              }
              onConfirm={remove}
              trigger={
                <Button colorPalette="red" variant="outline" size="xs">
                  <FiTrash2 />
                  Delete file
                </Button>
              }
            />
            <Button size="xs" variant="subtle" onClick={onClose}>
              Close
            </Button>
          </DrawerFooter>
        )}
        <DrawerCloseTrigger />
      </DrawerContent>
    </DrawerRoot>
  );
}

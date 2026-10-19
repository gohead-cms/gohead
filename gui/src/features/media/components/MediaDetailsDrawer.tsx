import { type ReactNode, useState } from "react";
import {
  Alert,
  AlertIcon,
  Box,
  Button,
  Center,
  Code,
  Divider,
  Drawer,
  DrawerBody,
  DrawerCloseButton,
  DrawerContent,
  DrawerFooter,
  DrawerHeader,
  DrawerOverlay,
  FormControl,
  FormLabel,
  Grid,
  HStack,
  IconButton,
  Image,
  Input,
  Link,
  Spinner,
  Text,
  Tooltip,
  useClipboard,
} from "@chakra-ui/react";
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
    return <Box as="video" src={src} controls preload="metadata" w="100%" maxH="320px" borderRadius="md" bg="black" />;
  }
  if (item.mime.startsWith("audio/")) {
    return (
      <Box>
        <MediaThumb item={item} height="120px" />
        <Box as="audio" src={src} controls w="100%" mt={2} />
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
  const { onCopy, hasCopied } = useClipboard(url);
  return (
    <FormControl>
      <FormLabel fontSize="sm">URL</FormLabel>
      <HStack spacing={1}>
        <Input size="sm" value={url} isReadOnly fontFamily="mono" fontSize="xs" onFocus={(e) => e.target.select()} />
        <Tooltip label={hasCopied ? "Copied" : "Copy URL"}>
          <IconButton aria-label="Copy URL" size="sm" icon={hasCopied ? <FiCheck /> : <FiCopy />} onClick={onCopy} />
        </Tooltip>
        <IconButton as={Link} href={url} isExternal aria-label="Open file in a new tab" size="sm" icon={<FiExternalLink />} />
      </HStack>
    </FormControl>
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
  canWrite,
  onClose,
  onChanged,
}: {
  id: number | null;
  canWrite: boolean;
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
    <Drawer isOpen={id !== null} onClose={onClose} size="md" placement="right">
      <DrawerOverlay />
      <DrawerContent>
        <DrawerCloseButton />
        <DrawerHeader pr={12}>
          <Text noOfLines={1} title={item?.name}>
            {item?.name ?? "File details"}
          </Text>
        </DrawerHeader>
        <DrawerBody>
          {query.error && (
            <Alert status="error" borderRadius="md">
              <AlertIcon />
              {query.error}
            </Alert>
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
              <Divider />
              <MediaDetailsForm
                key={`${item.id}-${item.updatedAt}`}
                item={item}
                canWrite={canWrite}
                onSaved={() => {
                  query.refetch();
                  onChanged();
                }}
              />
              {deleteError?.id === item.id && (
                <Alert status="error" borderRadius="md" fontSize="sm">
                  <AlertIcon />
                  {deleteError.message}
                </Alert>
              )}
            </Box>
          )}
        </DrawerBody>
        {item && canWrite && (
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
                <Button leftIcon={<FiTrash2 />} colorScheme="red" variant="outline" size="sm">
                  Delete file
                </Button>
              }
            />
            <Button size="sm" onClick={onClose}>
              Close
            </Button>
          </DrawerFooter>
        )}
      </DrawerContent>
    </Drawer>
  );
}

import { Box, Center, Icon, Image, Text, VStack } from "@chakra-ui/react";
import { FiFile, FiFileText, FiFilm, FiMusic } from "react-icons/fi";
import type { IconType } from "react-icons";
import { type MediaItem, absoluteUrl, extension, isImage } from "../media";

function iconFor(mime: string): IconType {
  if (mime.startsWith("video/")) return FiFilm;
  if (mime.startsWith("audio/")) return FiMusic;
  if (mime.startsWith("text/") || mime === "application/pdf") return FiFileText;
  return FiFile;
}

/** An image thumbnail, or an icon with the file's extension. */
export function MediaThumb({ item, height }: { item: MediaItem; height: string | number }) {
  if (isImage(item.mime)) {
    return (
      <Box h={height} bg="gray.100" overflow="hidden">
        <Image
          src={absoluteUrl(item.url)}
          alt={item.alt || item.name}
          w="100%"
          h="100%"
          objectFit="cover"
          loading="lazy"
          fallback={<Center h="100%"><Icon as={FiFile} boxSize={8} color="gray.400" /></Center>}
        />
      </Box>
    );
  }
  return (
    <Center h={height} bg="gray.50">
      <VStack spacing={1}>
        <Icon as={iconFor(item.mime)} boxSize={8} color="gray.400" />
        <Text fontSize="xs" fontWeight="bold" color="gray.500" letterSpacing="wide">
          {extension(item.name, item.mime)}
        </Text>
      </VStack>
    </Center>
  );
}

import React, { useRef, useState } from "react";
import { Box, Button, HStack, IconButton, Image, Input, Text, VStack, useDisclosure, useToast } from "@chakra-ui/react";
import { FiUpload, FiX, FiFile, FiImage } from "react-icons/fi";
import { API_BASE_URL, apiFetchWithAuth } from "../../../services/api";
import { useMe } from "../../auth/me";
import { MEDIA } from "../../auth/permissions";
import { MediaLibraryPicker } from "./MediaLibraryPicker";

// A media value as the API returns it (populated), or a bare ID / URL.
export interface MediaFile {
  id: number;
  url: string;
  name?: string;
  mime?: string;
}
export type MediaValue = MediaFile | number | string | null | undefined;

interface MediaFieldProps {
  id: string;
  value: MediaValue | MediaValue[];
  multiple?: boolean;
  accept?: string[];
  onChange: (value: MediaValue | MediaValue[]) => void;
}

// Relative URLs (/uploads/...) point at the API, not at the admin UI.
const absolute = (url: string) => (url.startsWith("/") ? `${API_BASE_URL}${url}` : url);

function Preview({ value, onRemove }: { value: MediaValue; onRemove: () => void }) {
  if (value === null || value === undefined) return null;
  const file = typeof value === "object" ? value : null;
  const url = file ? file.url : typeof value === "string" ? value : "";
  const label = file?.name || (typeof value === "number" ? `File #${value}` : url);
  const isImage = file ? file.mime?.startsWith("image/") : /\.(png|jpe?g|gif|webp|avif)$/i.test(url);
  return (
    <HStack borderWidth="1px" borderRadius="md" p={2} spacing={3} w="100%">
      {isImage && url ? (
        <Image src={absolute(url)} alt={label} boxSize="48px" objectFit="cover" borderRadius="sm" />
      ) : (
        <Box as={FiFile} boxSize="24px" color="gray.500" />
      )}
      <Text flex="1" fontSize="sm" noOfLines={1}>{label}</Text>
      <IconButton aria-label="Remove file" icon={<FiX />} size="xs" variant="ghost" onClick={onRemove} />
    </HStack>
  );
}

// MediaField uploads files to POST /api/media, or picks files already in the
// library, and stores them in the form; saving the entry sends their IDs.
export function MediaField({ id, value, multiple, accept, onChange }: MediaFieldProps) {
  const input = useRef<HTMLInputElement>(null);
  const [uploading, setUploading] = useState(false);
  const toast = useToast();
  // Uploading needs media create; files already set can still be removed.
  const me = useMe();
  const canUpload = me.can("create", MEDIA);
  const canBrowse = me.can("read", MEDIA);
  const library = useDisclosure();
  const values: MediaValue[] = multiple ? (Array.isArray(value) ? value : []) : value ? [value as MediaValue] : [];

  const upload = async (files: FileList | null) => {
    if (!files || files.length === 0) return;
    const body = new FormData();
    Array.from(files).forEach((f) => body.append("files", f));
    setUploading(true);
    try {
      const res = await apiFetchWithAuth("/api/media", { method: "POST", body });
      const json = await res.json();
      if (!res.ok) throw new Error(json.error?.message || "Upload failed");
      const uploaded: MediaFile[] = json.data;
      onChange(multiple ? [...values, ...uploaded] : uploaded[0]);
    } catch (err) {
      toast({ title: "Upload failed", description: (err as Error).message, status: "error", duration: 5000, isClosable: true });
    } finally {
      setUploading(false);
      if (input.current) input.current.value = "";
    }
  };

  const removeAt = (i: number) => {
    if (multiple) onChange(values.filter((_, j) => j !== i));
    else onChange(null);
  };

  return (
    <VStack align="stretch" spacing={2}>
      {values.map((v, i) => (
        <Preview key={i} value={v} onRemove={() => removeAt(i)} />
      ))}
      {!canUpload && !canBrowse && values.length === 0 && (
        <Text fontSize="sm" color="gray.500">
          No file. Your role cannot upload or browse files.
        </Text>
      )}
      {(multiple || values.length === 0) && (canUpload || canBrowse) && (
        <HStack spacing={2} wrap="wrap">
          {canUpload && (
            <Button leftIcon={<FiUpload />} size="sm" variant="outline" isLoading={uploading} onClick={() => input.current?.click()}>
              {multiple ? "Upload files" : "Upload a file"}
            </Button>
          )}
          {canBrowse && (
            <Button leftIcon={<FiImage />} size="sm" variant="outline" onClick={library.onOpen}>
              Choose from library
            </Button>
          )}
        </HStack>
      )}
      {library.isOpen && (
        <MediaLibraryPicker
          multiple={multiple}
          accept={accept}
          selected={values.map((v) => (v && typeof v === "object" ? v.id : typeof v === "number" ? v : -1))}
          onClose={library.onClose}
          onPick={(files) => {
            onChange(multiple ? [...values, ...files] : (files[0] ?? null));
            library.onClose();
          }}
        />
      )}
      <Input
        ref={input}
        id={id}
        type="file"
        display="none"
        multiple={multiple}
        accept={accept?.join(",")}
        onChange={(e: React.ChangeEvent<HTMLInputElement>) => upload(e.target.files)}
      />
    </VStack>
  );
}

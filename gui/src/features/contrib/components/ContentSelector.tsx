import { Box, Spinner, Text, VStack, chakra } from "@chakra-ui/react";
import { useColorModeValue } from "@/components/ui/color-mode";
import type { Schema } from "../../../shared/types";
import type { SingletonDef } from "../hooks/useSingletons";
import type { Selection } from "./selection";

interface ContentSelectorProps {
  collections: { schema: Schema }[];
  singletons: SingletonDef[];
  /** False hides the single types (the list is admin-only). */
  showSingletons: boolean;
  selected: Selection | null;
  onSelect: (selection: Selection) => void;
  collectionsLoading: boolean;
  singletonsLoading: boolean;
  singletonsError: string | null;
}

function SelectorItem({ label, active, onClick }: { label: string; active: boolean; onClick: () => void }) {
  const activeBg = useColorModeValue("purple.50", "purple.900");
  const activeColor = useColorModeValue("purple.700", "white");
  const hoverBg = useColorModeValue("gray.100", "gray.700");
  return (
    <chakra.button
      type="button"
      onClick={onClick}
      aria-current={active ? "true" : undefined}
      w="full"
      textAlign="left"
      py={2}
      px={3}
      borderRadius="md"
      bg={active ? activeBg : "transparent"}
      color={active ? activeColor : "inherit"}
      fontWeight={active ? "semibold" : "normal"}
      _hover={{ bg: active ? activeBg : hoverBg }}
      lineClamp={1}
    >
      {label}
    </chakra.button>
  );
}

function SectionTitle({ children }: { children: string }) {
  return (
    <Text fontSize="xs" fontWeight="bold" textTransform="uppercase" letterSpacing="wide" color="gray.500" mb={2} px={3}>
      {children}
    </Text>
  );
}

/** The collections and single types whose content can be browsed. */
export function ContentSelector({
  collections,
  singletons,
  showSingletons,
  selected,
  onSelect,
  collectionsLoading,
  singletonsLoading,
  singletonsError,
}: ContentSelectorProps) {
  const sidebarBg = useColorModeValue("white", "gray.800");
  const borderColor = useColorModeValue("gray.200", "gray.700");
  const isActive = (kind: Selection["kind"], name: string) => selected?.kind === kind && selected.name === name;

  return (
    <Box w={{ base: "200px", xl: "260px" }} flexShrink={0} bg={sidebarBg} borderRight="1px solid" borderColor={borderColor} p={4} overflowY="auto">
      <SectionTitle>Collections</SectionTitle>
      {collectionsLoading ? (
        <Spinner size="sm" mx={3} />
      ) : collections.length === 0 ? (
        <Text fontSize="sm" color="gray.400" px={3}>
          No collections.
        </Text>
      ) : (
        <VStack gap={1} align="stretch">
          {collections.map(({ schema }) => {
            const name = schema.collectionName ?? "";
            return (
              <SelectorItem
                key={name}
                label={schema.info?.displayName || name}
                active={isActive("collection", name)}
                onClick={() => onSelect({ kind: "collection", name })}
              />
            );
          })}
        </VStack>
      )}

      {showSingletons && (
        <Box mt={6}>
          <SectionTitle>Single types</SectionTitle>
          {singletonsLoading ? (
            <Spinner size="sm" mx={3} />
          ) : singletonsError ? (
            <Text fontSize="sm" color="red.500" px={3}>
              {singletonsError}
            </Text>
          ) : singletons.length === 0 ? (
            <Text fontSize="sm" color="gray.400" px={3}>
              No single types.
            </Text>
          ) : (
            <VStack gap={1} align="stretch">
              {singletons.map((s) => (
                <SelectorItem
                  key={s.name}
                  label={s.name}
                  active={isActive("singleton", s.name)}
                  onClick={() => onSelect({ kind: "singleton", name: s.name })}
                />
              ))}
            </VStack>
          )}
        </Box>
      )}
    </Box>
  );
}

import {
  Badge,
  Box,
  Button,
  Checkbox,
  HStack,
  Menu,
  MenuButton,
  MenuDivider,
  MenuItem,
  MenuList,
  Table,
  TableContainer,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
} from "@chakra-ui/react";
import { FiChevronDown } from "react-icons/fi";
import {
  ANY_TYPE,
  COLLECTION_ACTIONS,
  type ContentAction,
  MEDIA_ACTIONS,
  SINGLETON_ACTIONS,
  singletonResource,
  sortActions,
  toggleAction,
} from "../../auth/permissions";
import { MEDIA_PRESETS, PRESETS } from "./roles";

/** The matrix's value: the role's content map (without null) and media actions. */
export interface MatrixValue {
  content: Record<string, string[]>;
  media: string[];
}

type RowKind = "any" | "collection" | "singleton";

interface Row {
  key: string;
  label: string;
  kind: RowKind;
}

const ACTION_LABELS: Record<string, string> = {
  read: "Read",
  create: "Create",
  update: "Update",
  delete: "Delete",
  publish: "Publish",
};

function applicable(kind: RowKind | "media", columns: readonly string[]): readonly string[] {
  const all: readonly string[] = kind === "media" ? MEDIA_ACTIONS : kind === "singleton" ? SINGLETON_ACTIONS : COLLECTION_ACTIONS;
  return all.filter((a) => columns.includes(a));
}

/**
 * Edits a role's permissions: a row per content type (plus "all other
 * types" and the media library), a column per action. A type without its
 * own row uses "all other types" until one of its boxes is changed.
 * actions limits the columns (and presets) to some actions, e.g. read only
 * for the public role.
 */
export function PermissionMatrix({
  value,
  onChange,
  collections,
  singletons,
  isReadOnly,
  actions: columns = COLLECTION_ACTIONS,
}: {
  value: MatrixValue;
  onChange: (v: MatrixValue) => void;
  collections: string[];
  singletons: string[];
  isReadOnly?: boolean;
  actions?: readonly ContentAction[];
}) {
  const fits = (actions: readonly string[]) => actions.every((a) => (columns as readonly string[]).includes(a));
  const presets = PRESETS.filter((p) => fits(p.collection) && fits(p.singleton));
  const mediaPresets = MEDIA_PRESETS.filter((p) => fits(p.actions));
  const mediaActions = applicable("media", columns);
  const anyActions = value.content[ANY_TYPE] ?? [];
  const rows: Row[] = [
    ...collections.map((c): Row => ({ key: c, label: c, kind: "collection" })),
    ...singletons.map((s): Row => ({ key: singletonResource(s), label: s, kind: "singleton" })),
  ];

  /** The actions a row has: its own, or those of "*" that apply to it. */
  const effective = (row: Row): string[] => {
    const own = value.content[row.key];
    const actions = own ?? anyActions;
    const allowed = applicable(row.kind, columns);
    return actions.filter((a) => allowed.includes(a));
  };
  const inherited = (row: Row) => row.kind !== "any" && !(row.key in value.content);

  const setRow = (key: string, actions: string[] | undefined) => {
    const content = { ...value.content };
    if (actions === undefined) delete content[key];
    else content[key] = sortActions(actions);
    onChange({ ...value, content });
  };

  const anyRow: Row = { key: ANY_TYPE, label: "All other types", kind: "any" };

  const renderRow = (row: Row) => {
    const actions = row.kind === "any" ? anyActions : effective(row);
    const isInherited = inherited(row);
    const allowed = applicable(row.kind, columns);
    return (
      <Tr key={row.key} data-testid={`perm-row-${row.key}`}>
        <Td>
          <Text fontWeight={row.kind === "any" ? "semibold" : "medium"}>
            {row.label}
            {row.kind === "any" && (
              <Text as="span" color="gray.500" fontWeight="normal">
                {" "}
                (*)
              </Text>
            )}
          </Text>
          {row.kind === "any" ? (
            <Text fontSize="xs" color="gray.500">
              Every type without its own settings
            </Text>
          ) : (
            isInherited && (
              <Text fontSize="xs" color="gray.500">
                Same as all other types
              </Text>
            )
          )}
        </Td>
        {columns.map((a) => (
          <Td key={a} textAlign="center">
            {allowed.includes(a) ? (
              <Checkbox
                colorScheme="purple"
                aria-label={`${ACTION_LABELS[a]} ${row.label}`}
                isChecked={actions.includes(a)}
                isReadOnly={isReadOnly}
                opacity={isInherited ? 0.55 : 1}
                onChange={(e) => setRow(row.key, toggleAction(actions, a, e.target.checked))}
              />
            ) : (
              <Text color="gray.300" aria-hidden>
                –
              </Text>
            )}
          </Td>
        ))}
        {!isReadOnly && (
          <Td textAlign="right">
            <Menu placement="bottom-end" isLazy>
              <MenuButton as={Button} size="xs" variant="ghost" rightIcon={<FiChevronDown />} aria-label={`Presets for ${row.label}`}>
                Preset
              </MenuButton>
              <MenuList fontSize="sm">
                {presets.map((p) => (
                  <MenuItem key={p.label} onClick={() => setRow(row.key, row.kind === "singleton" ? p.singleton : p.collection)}>
                    {p.label}
                  </MenuItem>
                ))}
                {row.kind !== "any" && (
                  <>
                    <MenuDivider />
                    <MenuItem isDisabled={isInherited} onClick={() => setRow(row.key, undefined)}>
                      Same as all other types
                    </MenuItem>
                  </>
                )}
              </MenuList>
            </Menu>
          </Td>
        )}
      </Tr>
    );
  };

  const section = (title: string, hint?: string) => (
    <Tr bg="gray.50">
      <Td colSpan={columns.length + (isReadOnly ? 1 : 2)} py={2}>
        <HStack spacing={2}>
          <Text fontSize="xs" fontWeight="bold" textTransform="uppercase" letterSpacing="wide" color="gray.600">
            {title}
          </Text>
          {hint && (
            <Text fontSize="xs" color="gray.500">
              {hint}
            </Text>
          )}
        </HStack>
      </Td>
    </Tr>
  );

  const collectionRows = rows.filter((r) => r.kind === "collection");
  const singletonRows = rows.filter((r) => r.kind === "singleton");

  return (
    <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
      <TableContainer>
        <Table size="sm">
          <Thead>
            <Tr>
              <Th>Type</Th>
              {columns.map((a) => (
                <Th key={a} textAlign="center" px={2}>
                  {ACTION_LABELS[a]}
                </Th>
              ))}
              {!isReadOnly && <Th />}
            </Tr>
          </Thead>
          <Tbody>
            {renderRow(anyRow)}
            {collectionRows.length > 0 && section("Collections")}
            {collectionRows.map(renderRow)}
            {singletonRows.length > 0 && section("Single types", "read and update only")}
            {singletonRows.map(renderRow)}
            {section("Media library")}
            <Tr data-testid="perm-row-media">
              <Td>
                <Text fontWeight="medium">Media</Text>
                <Text fontSize="xs" color="gray.500">
                  {mediaActions.length === MEDIA_ACTIONS.length ? "Browse, upload, edit and delete files" : "Browse files"}
                </Text>
              </Td>
              {columns.map((a) => (
                <Td key={a} textAlign="center">
                  {mediaActions.includes(a) ? (
                    <Checkbox
                      colorScheme="purple"
                      aria-label={`${ACTION_LABELS[a]} media`}
                      isChecked={value.media.includes(a)}
                      isReadOnly={isReadOnly}
                      onChange={(e) => onChange({ ...value, media: sortActions(toggleAction(value.media, a, e.target.checked), MEDIA_ACTIONS) })}
                    />
                  ) : (
                    <Text color="gray.300" aria-hidden>
                      –
                    </Text>
                  )}
                </Td>
              ))}
              {!isReadOnly && (
                <Td textAlign="right">
                  <Menu placement="bottom-end" isLazy>
                    <MenuButton as={Button} size="xs" variant="ghost" rightIcon={<FiChevronDown />} aria-label="Presets for media">
                      Preset
                    </MenuButton>
                    <MenuList fontSize="sm">
                      {mediaPresets.map((p) => (
                        <MenuItem key={p.label} onClick={() => onChange({ ...value, media: [...p.actions] })}>
                          {p.label}
                        </MenuItem>
                      ))}
                    </MenuList>
                  </Menu>
                </Td>
              )}
            </Tr>
          </Tbody>
        </Table>
      </TableContainer>
      {(collections.length > 0 || singletons.length > 0) && (
        <HStack px={4} py={2} borderTopWidth="1px" spacing={2} flexWrap="wrap">
          <Badge colorScheme="gray" variant="subtle">
            Tip
          </Badge>
          <Text fontSize="xs" color="gray.500">
            {columns.length > 1 && "Any action includes read: checking one checks Read, unchecking Read clears the row. "}
            Faded rows follow “All other types”.
          </Text>
        </HStack>
      )}
    </Box>
  );
}

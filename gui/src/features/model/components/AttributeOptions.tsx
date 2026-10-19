import {
  Checkbox,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  Select,
  SimpleGrid,
  Textarea,
} from "@chakra-ui/react";
import type { AttributeField, AttributeForm } from "../schema";
import {
  minMaxLabels,
  RELATION_TYPES,
  supportsMinMax,
  supportsPattern,
} from "../schema";

export interface AttributeOptionsProps {
  attr: AttributeForm;
  errors: Partial<Record<AttributeField, string>>;
  onChange: (patch: Partial<AttributeForm>) => void;
  /** Collections a relation can target. */
  collections: string[];
  /** Components a component attribute can use. */
  components: string[];
}

/** The type-specific settings of an attribute. */
export function AttributeOptions({ attr, errors, onChange, collections, components }: AttributeOptionsProps) {
  return (
    <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} spacing={3}>
      {attr.type === "enum" && (
        <FormControl isInvalid={!!errors.options} isRequired gridColumn={{ md: "span 2", xl: "span 3" }}>
          <FormLabel fontSize="sm">Options</FormLabel>
          <Textarea
            size="sm"
            rows={3}
            value={attr.options.join("\n")}
            onChange={(e) => onChange({ options: e.target.value.split("\n") })}
            placeholder={"draft\nactive\narchived"}
          />
          {errors.options ? <FormErrorMessage>{errors.options}</FormErrorMessage> : <FormHelperText>One option per line.</FormHelperText>}
        </FormControl>
      )}

      {attr.type === "relation" && (
        <>
          <FormControl isInvalid={!!errors.target} isRequired>
            <FormLabel fontSize="sm">Target collection</FormLabel>
            <Select size="sm" value={attr.target} onChange={(e) => onChange({ target: e.target.value })} placeholder="Choose a collection">
              {withCurrent(collections, attr.target).map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </Select>
            <FormErrorMessage>{errors.target}</FormErrorMessage>
          </FormControl>
          <FormControl isRequired>
            <FormLabel fontSize="sm">Relation type</FormLabel>
            <Select size="sm" value={attr.relation} onChange={(e) => onChange({ relation: e.target.value })}>
              {RELATION_TYPES.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </Select>
          </FormControl>
        </>
      )}

      {attr.type === "component" && (
        <>
          <FormControl isInvalid={!!errors.component} isRequired>
            <FormLabel fontSize="sm">Component</FormLabel>
            <Select
              size="sm"
              value={attr.component}
              onChange={(e) => onChange({ component: e.target.value })}
              placeholder="Choose a component"
            >
              {withCurrent(components, attr.component).map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </Select>
            {errors.component ? (
              <FormErrorMessage>{errors.component}</FormErrorMessage>
            ) : (
              components.length === 0 && <FormHelperText>Create a component first.</FormHelperText>
            )}
          </FormControl>
          <FormControl alignSelf="end" pb={1}>
            <Checkbox isChecked={attr.repeatable} onChange={(e) => onChange({ repeatable: e.target.checked })}>
              Repeatable (a list)
            </Checkbox>
          </FormControl>
        </>
      )}

      {attr.type === "media" && (
        <>
          <FormControl>
            <FormLabel fontSize="sm">Allowed types</FormLabel>
            <Input
              size="sm"
              value={attr.allowedTypes}
              onChange={(e) => onChange({ allowedTypes: e.target.value })}
              placeholder="image/*, application/pdf"
            />
            <FormHelperText>Comma-separated MIME patterns; empty allows any file.</FormHelperText>
          </FormControl>
          <FormControl alignSelf="center">
            <Checkbox isChecked={attr.multiple} onChange={(e) => onChange({ multiple: e.target.checked })}>
              Multiple files
            </Checkbox>
          </FormControl>
        </>
      )}

      {supportsPattern(attr) && (
        <FormControl isInvalid={!!errors.pattern}>
          <FormLabel fontSize="sm">Pattern</FormLabel>
          <Input
            size="sm"
            fontFamily="mono"
            value={attr.pattern}
            onChange={(e) => onChange({ pattern: e.target.value })}
            placeholder="^[a-z0-9-]+$"
          />
          {errors.pattern ? <FormErrorMessage>{errors.pattern}</FormErrorMessage> : <FormHelperText>Regular expression values must match.</FormHelperText>}
        </FormControl>
      )}

      {supportsMinMax(attr) && <MinMax attr={attr} errors={errors} onChange={onChange} />}
    </SimpleGrid>
  );
}

function MinMax({ attr, errors, onChange }: Pick<AttributeOptionsProps, "attr" | "errors" | "onChange">) {
  const labels = minMaxLabels(attr);
  return (
    <>
      <FormControl isInvalid={!!errors.min}>
        <FormLabel fontSize="sm">{labels.min}</FormLabel>
        <Input size="sm" type="number" step={1} value={attr.min} onChange={(e) => onChange({ min: e.target.value })} />
        <FormErrorMessage>{errors.min}</FormErrorMessage>
      </FormControl>
      <FormControl isInvalid={!!errors.max}>
        <FormLabel fontSize="sm">{labels.max}</FormLabel>
        <Input size="sm" type="number" step={1} value={attr.max} onChange={(e) => onChange({ max: e.target.value })} />
        <FormErrorMessage>{errors.max}</FormErrorMessage>
      </FormControl>
    </>
  );
}

/** Keeps a saved value selectable even if it is not in the loaded list. */
function withCurrent(list: string[], current: string): string[] {
  return current && !list.includes(current) ? [...list, current] : list;
}

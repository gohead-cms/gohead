import { Field, Input, NativeSelect, SimpleGrid, Textarea } from "@chakra-ui/react";
import { Checkbox } from "@/components/ui/checkbox";
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
    <SimpleGrid columns={{ base: 1, md: 2, xl: 3 }} gap={3}>
      {attr.type === "enum" && (
        <Field.Root invalid={!!errors.options} required gridColumn={{ md: "span 2", xl: "span 3" }}>
          <Field.Label fontSize="sm">
            Options
            <Field.RequiredIndicator />
          </Field.Label>
          <Textarea
            size="sm"
            rows={3}
            value={attr.options.join("\n")}
            onChange={(e) => onChange({ options: e.target.value.split("\n") })}
            placeholder={"draft\nactive\narchived"}
          />
          {errors.options ? <Field.ErrorText>{errors.options}</Field.ErrorText> : <Field.HelperText>One option per line.</Field.HelperText>}
        </Field.Root>
      )}

      {attr.type === "relation" && (
        <>
          <Field.Root invalid={!!errors.target} required>
            <Field.Label fontSize="sm">
              Target collection
              <Field.RequiredIndicator />
            </Field.Label>
            <NativeSelect.Root size="xs">
              <NativeSelect.Field value={attr.target} onChange={(e) => onChange({ target: e.target.value })} placeholder="Choose a collection">
                {withCurrent(collections, attr.target).map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </NativeSelect.Field>
              <NativeSelect.Indicator />
            </NativeSelect.Root>
            <Field.ErrorText>{errors.target}</Field.ErrorText>
          </Field.Root>
          <Field.Root required>
            <Field.Label fontSize="sm">
              Relation type
              <Field.RequiredIndicator />
            </Field.Label>
            <NativeSelect.Root size="xs">
              <NativeSelect.Field value={attr.relation} onChange={(e) => onChange({ relation: e.target.value })}>
                {RELATION_TYPES.map((r) => (
                  <option key={r.value} value={r.value}>
                    {r.label}
                  </option>
                ))}
              </NativeSelect.Field>
              <NativeSelect.Indicator />
            </NativeSelect.Root>
          </Field.Root>
        </>
      )}

      {attr.type === "component" && (
        <>
          <Field.Root invalid={!!errors.component} required>
            <Field.Label fontSize="sm">
              Component
              <Field.RequiredIndicator />
            </Field.Label>
            <NativeSelect.Root size="xs">
              <NativeSelect.Field
                value={attr.component}
                onChange={(e) => onChange({ component: e.target.value })}
                placeholder="Choose a component"
              >
                {withCurrent(components, attr.component).map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </NativeSelect.Field>
              <NativeSelect.Indicator />
            </NativeSelect.Root>
            {errors.component ? (
              <Field.ErrorText>{errors.component}</Field.ErrorText>
            ) : (
              components.length === 0 && <Field.HelperText>Create a component first.</Field.HelperText>
            )}
          </Field.Root>
          <Field.Root alignSelf="end" pb={1}>
            <Checkbox size="sm" checked={attr.repeatable} onCheckedChange={(e) => onChange({ repeatable: !!e.checked })}>
              Repeatable (a list)
            </Checkbox>
          </Field.Root>
        </>
      )}

      {attr.type === "media" && (
        <>
          <Field.Root>
            <Field.Label fontSize="sm">Allowed types</Field.Label>
            <Input
              size="xs"
              value={attr.allowedTypes}
              onChange={(e) => onChange({ allowedTypes: e.target.value })}
              placeholder="image/*, application/pdf"
            />
            <Field.HelperText>Comma-separated MIME patterns; empty allows any file.</Field.HelperText>
          </Field.Root>
          <Field.Root alignSelf="center">
            <Checkbox size="sm" checked={attr.multiple} onCheckedChange={(e) => onChange({ multiple: !!e.checked })}>
              Multiple files
            </Checkbox>
          </Field.Root>
        </>
      )}

      {supportsPattern(attr) && (
        <Field.Root invalid={!!errors.pattern}>
          <Field.Label fontSize="sm">Pattern</Field.Label>
          <Input
            size="xs"
            fontFamily="mono"
            value={attr.pattern}
            onChange={(e) => onChange({ pattern: e.target.value })}
            placeholder="^[a-z0-9-]+$"
          />
          {errors.pattern ? <Field.ErrorText>{errors.pattern}</Field.ErrorText> : <Field.HelperText>Regular expression values must match.</Field.HelperText>}
        </Field.Root>
      )}

      {supportsMinMax(attr) && <MinMax attr={attr} errors={errors} onChange={onChange} />}
    </SimpleGrid>
  );
}

function MinMax({ attr, errors, onChange }: Pick<AttributeOptionsProps, "attr" | "errors" | "onChange">) {
  const labels = minMaxLabels(attr);
  return (
    <>
      <Field.Root invalid={!!errors.min}>
        <Field.Label fontSize="sm">{labels.min}</Field.Label>
        <Input size="xs" type="number" step={1} value={attr.min} onChange={(e) => onChange({ min: e.target.value })} />
        <Field.ErrorText>{errors.min}</Field.ErrorText>
      </Field.Root>
      <Field.Root invalid={!!errors.max}>
        <Field.Label fontSize="sm">{labels.max}</Field.Label>
        <Input size="xs" type="number" step={1} value={attr.max} onChange={(e) => onChange({ max: e.target.value })} />
        <Field.ErrorText>{errors.max}</Field.ErrorText>
      </Field.Root>
    </>
  );
}

/** Keeps a saved value selectable even if it is not in the loaded list. */
function withCurrent(list: string[], current: string): string[] {
  return current && !list.includes(current) ? [...list, current] : list;
}

import React, { type ReactNode } from "react";
import { Field, Input, NativeSelect, Text, Textarea, VStack } from "@chakra-ui/react";
import { Switch } from "@/components/ui/switch";
import type { Attribute } from "../../../shared/types";
import { ComponentField } from "./ComponentField";
import { JsonField } from "./JsonField";
import { MediaField, type MediaValue } from "./MediaField";
import { RelationField } from "./RelationField";
import { relationTarget } from "../../model/schema";
import { inputType, isBooleanType } from "./entryValues";

type ChangeHandler = (key: string, value: unknown) => void;

interface FieldProps {
  id: string;
  name: string;
  attr: Attribute;
  value: unknown;
  onChange: ChangeHandler;
  placeholder?: string;
}

function FieldInput({ id, name, attr, value, onChange, placeholder = `Enter ${name}` }: FieldProps) {
  const text = typeof value === "string" || typeof value === "number" ? value : "";
  const onText = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => onChange(name, e.target.value);

  if (isBooleanType(attr.type)) {
    return <Switch size="sm" inputProps={{ id }} checked={value === true} onCheckedChange={(e) => onChange(name, e.checked)} />;
  }
  if (attr.type === "enum" && attr.options?.length) {
    return (
      <NativeSelect.Root>
        <NativeSelect.Field id={id} value={typeof value === "string" ? value : ""} placeholder="Choose…" onChange={(e) => onChange(name, e.target.value)}>
          {attr.options.map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </NativeSelect.Field>
        <NativeSelect.Indicator />
      </NativeSelect.Root>
    );
  }
  switch (attr.type) {
    case "component":
      return <ComponentField id={id} name={name} attr={attr} value={value} onChange={(v) => onChange(name, v)} />;
    case "relation":
      return <RelationField id={id} name={name} attr={attr} value={value} onChange={(v) => onChange(name, v)} />;
    case "json":
      return <JsonField id={id} value={value} onChange={(v) => onChange(name, v)} />;
    case "media":
      return (
        <MediaField
          id={id}
          name={name}
          value={value as MediaValue | MediaValue[]}
          multiple={attr.multiple}
          accept={attr.allowed_types}
          onChange={(v) => onChange(name, v)}
        />
      );
    case "text":
      return <Textarea id={id} value={text} onChange={onText} placeholder={placeholder} />;
    case "richtext":
      return <Textarea id={id} value={text} onChange={onText} placeholder={placeholder} rows={8} />;
    default:
      return <Input id={id} type={inputType(attr.type)} value={text} onChange={onText} placeholder={placeholder} />;
  }
}

// Fields made of several controls: their label names a group, not an input.
const GROUPS = new Set(["relation", "component", "media"]);

interface EntryFieldsProps {
  attributes: Record<string, Attribute>;
  values: Record<string, unknown>;
  onChange: ChangeHandler;
  /** Prefix of the input ids, to keep them unique on the page. */
  idPrefix?: string;
  /** Help shown under a field, by attribute name (e.g. the default locale's value). */
  hints?: Record<string, ReactNode>;
  /** Placeholders of text fields, by attribute name. */
  placeholders?: Record<string, string>;
}

/**
 * One labelled input per attribute of a collection, singleton or component:
 * relations pick entries, components nest their fields, json is edited as
 * text.
 */
export function EntryFields({ attributes, values, onChange, idPrefix = "field-", hints, placeholders }: EntryFieldsProps) {
  return (
    <VStack gap={6} align="stretch">
      {Object.entries(attributes).map(([key, attr]) => {
        const id = `${idPrefix}${key}`;
        // A group's label names the group (aria-labelledby), not an input:
        // the field keeps its own generated control id.
        return (
          <Field.Root key={key} required={attr.required} id={GROUPS.has(attr.type) ? undefined : id} ids={{ label: `${id}-label` }}>
            <Field.Label>
              {key}
              {attr.type === "relation" && attr.target && (
                <Text as="span" fontWeight="normal" color="gray.500" fontSize="sm">
                  {" "}
                  → {relationTarget(attr.target)}
                  {attr.relation === "manyToMany" ? " (several)" : ""}
                </Text>
              )}
              <Field.RequiredIndicator />
            </Field.Label>
            <FieldInput id={id} name={key} attr={attr} value={values[key]} onChange={onChange} placeholder={placeholders?.[key]} />
            {hints?.[key] && <Field.HelperText>{hints[key]}</Field.HelperText>}
          </Field.Root>
        );
      })}
    </VStack>
  );
}

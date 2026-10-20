import React, { type ReactNode } from "react";
import { FormControl, FormHelperText, FormLabel, Input, Select, Switch, Text, Textarea, VStack } from "@chakra-ui/react";
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
    return <Switch id={id} isChecked={value === true} onChange={(e) => onChange(name, e.target.checked)} />;
  }
  if (attr.type === "enum" && attr.options?.length) {
    return (
      <Select id={id} value={typeof value === "string" ? value : ""} placeholder="Choose…" onChange={(e) => onChange(name, e.target.value)}>
        {attr.options.map((o) => (
          <option key={o} value={o}>
            {o}
          </option>
        ))}
      </Select>
    );
  }
  switch (attr.type) {
    case "component":
      return <ComponentField id={id} attr={attr} value={value} onChange={(v) => onChange(name, v)} />;
    case "relation":
      return <RelationField id={id} attr={attr} value={value} onChange={(v) => onChange(name, v)} />;
    case "json":
      return <JsonField id={id} value={value} onChange={(v) => onChange(name, v)} />;
    case "media":
      return (
        <MediaField
          id={id}
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
    <VStack spacing={6} align="stretch">
      {Object.entries(attributes).map(([key, attr]) => {
        const id = `${idPrefix}${key}`;
        return (
          <FormControl key={key} isRequired={attr.required}>
            <FormLabel htmlFor={id}>
              {key}
              {attr.type === "relation" && attr.target && (
                <Text as="span" fontWeight="normal" color="gray.500" fontSize="sm">
                  {" "}
                  → {relationTarget(attr.target)}
                  {attr.relation === "manyToMany" ? " (several)" : ""}
                </Text>
              )}
            </FormLabel>
            <FieldInput id={id} name={key} attr={attr} value={values[key]} onChange={onChange} placeholder={placeholders?.[key]} />
            {hints?.[key] && <FormHelperText>{hints[key]}</FormHelperText>}
          </FormControl>
        );
      })}
    </VStack>
  );
}

import React, { type ReactNode } from "react";
import { FormControl, FormHelperText, FormLabel, Input, Select, Switch, Textarea, VStack } from "@chakra-ui/react";
import type { Attribute } from "../../../shared/types";
import { MediaField, type MediaValue } from "./MediaField";
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
      return (
        <Textarea id={id} value={value === undefined || value === null ? "" : JSON.stringify(value, null, 2)} isReadOnly rows={4} fontFamily="mono" fontSize="sm" />
      );
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
 * One labelled input per attribute of a collection or singleton. Relations
 * are not editable here; component values are shown as they are.
 */
export function EntryFields({ attributes, values, onChange, idPrefix = "field-", hints, placeholders }: EntryFieldsProps) {
  return (
    <VStack spacing={6} align="stretch">
      {Object.entries(attributes).map(([key, attr]) => {
        if (attr.type === "relation") return null;
        const id = `${idPrefix}${key}`;
        return (
          <FormControl key={key} isRequired={attr.required}>
            <FormLabel htmlFor={id}>{key}</FormLabel>
            <FieldInput id={id} name={key} attr={attr} value={values[key]} onChange={onChange} placeholder={placeholders?.[key]} />
            {attr.type === "component" && <FormHelperText>Component fields are edited through the API.</FormHelperText>}
            {hints?.[key] && <FormHelperText>{hints[key]}</FormHelperText>}
          </FormControl>
        );
      })}
    </VStack>
  );
}

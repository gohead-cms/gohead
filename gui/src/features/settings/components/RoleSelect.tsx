import { Field, NativeSelect } from "@chakra-ui/react";
import { BUILTIN_ROLE_HELP, isPublicRole, roleHelp, useRoles } from "./roles";

/**
 * Role picker (built-in and custom roles) with a one-line explanation of the
 * chosen role. The public role, for anonymous requests, is never offered.
 */
export function RoleSelect({
  value,
  onChange,
  isDisabled,
  disabledReason,
}: {
  value: string;
  onChange: (r: string) => void;
  isDisabled?: boolean;
  disabledReason?: string;
}) {
  const { data, error } = useRoles();
  const roles = (data ?? []).filter((r) => !isPublicRole(r));
  // Until the list arrives (or if it fails), the built-in roles and the
  // current value stay selectable.
  const names = roles.length ? roles.map((r) => r.name) : Object.keys(BUILTIN_ROLE_HELP);
  const options = value && !names.includes(value) ? [...names, value] : names;
  const custom = new Set(roles.filter((r) => !r.builtin).map((r) => r.name));
  const current = roles.find((r) => r.name === value);
  const help = error ? `Custom roles could not be loaded (${error}).` : current ? roleHelp(current) : (BUILTIN_ROLE_HELP[value] ?? "");

  return (
    <Field.Root disabled={isDisabled}>
      <Field.Label>Role</Field.Label>
      <NativeSelect.Root>
        <NativeSelect.Field value={value} onChange={(e) => onChange(e.target.value)}>
          {options.map((r) => (
            <option key={r} value={r}>
              {custom.has(r) ? `${r} (custom)` : r}
            </option>
          ))}
        </NativeSelect.Field>
        <NativeSelect.Indicator />
      </NativeSelect.Root>
      <Field.HelperText>{isDisabled && disabledReason ? disabledReason : help}</Field.HelperText>
    </Field.Root>
  );
}

import { FormControl, FormHelperText, FormLabel, Select } from "@chakra-ui/react";
import { BUILTIN_ROLE_HELP, roleHelp, useRoles } from "./roles";

/** Role picker (built-in and custom roles) with a one-line explanation of the chosen role. */
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
  const roles = data ?? [];
  // Until the list arrives (or if it fails), the built-in roles and the
  // current value stay selectable.
  const names = roles.length ? roles.map((r) => r.name) : Object.keys(BUILTIN_ROLE_HELP);
  const options = value && !names.includes(value) ? [...names, value] : names;
  const custom = new Set(roles.filter((r) => !r.builtin).map((r) => r.name));
  const current = roles.find((r) => r.name === value);
  const help = error
    ? `Custom roles could not be loaded (${error}).`
    : current
      ? roleHelp(current)
      : (BUILTIN_ROLE_HELP[value] ?? "");

  return (
    <FormControl isDisabled={isDisabled}>
      <FormLabel>Role</FormLabel>
      <Select value={value} onChange={(e) => onChange(e.target.value)}>
        {options.map((r) => (
          <option key={r} value={r}>
            {custom.has(r) ? `${r} (custom)` : r}
          </option>
        ))}
      </Select>
      <FormHelperText>{isDisabled && disabledReason ? disabledReason : help}</FormHelperText>
    </FormControl>
  );
}

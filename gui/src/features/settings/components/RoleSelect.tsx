import { FormControl, FormHelperText, FormLabel, Select } from "@chakra-ui/react";
import { ROLE_HELP, ROLES, type RoleName } from "./types";

/** Role picker with a one-line explanation of the chosen role. */
export function RoleSelect({
  value,
  onChange,
  isDisabled,
  disabledReason,
}: {
  value: RoleName;
  onChange: (r: RoleName) => void;
  isDisabled?: boolean;
  disabledReason?: string;
}) {
  return (
    <FormControl isDisabled={isDisabled}>
      <FormLabel>Role</FormLabel>
      <Select value={value} onChange={(e) => onChange(e.target.value as RoleName)}>
        {ROLES.map((r) => (
          <option key={r} value={r}>
            {r}
          </option>
        ))}
      </Select>
      <FormHelperText>{isDisabled && disabledReason ? disabledReason : ROLE_HELP[value]}</FormHelperText>
    </FormControl>
  );
}

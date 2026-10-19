import { HStack, Icon, Select, Tooltip } from "@chakra-ui/react";
import { FiGlobe } from "react-icons/fi";
import type { Locale } from "../../../shared/hooks/useLocales";

interface LocaleSwitcherProps {
  locales: Locale[];
  /** The selected locale code. */
  value: string;
  onChange: (code: string) => void;
  /** Locale codes that have content (an entry's "locales"): marked in the list. */
  translated?: string[];
  /** Why the switcher is disabled, shown as a tooltip. */
  disabledReason?: string;
  label?: string;
}

/** Picks the locale an entry is edited (or a list is read) in; the default locale comes first. */
export function LocaleSwitcher({ locales, value, onChange, translated, disabledReason, label = "Locale" }: LocaleSwitcherProps) {
  const select = (
    <HStack spacing={2} flexShrink={0}>
      <Icon as={FiGlobe} color="purple.500" aria-hidden />
      <Select
        size="sm"
        w="auto"
        bg="white"
        borderRadius="md"
        aria-label={label}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        isDisabled={!!disabledReason}
      >
        {locales.map((l) => (
          <option key={l.code} value={l.code}>
            {optionText(l, translated)}
          </option>
        ))}
      </Select>
    </HStack>
  );
  if (!disabledReason) return select;
  return (
    <Tooltip hasArrow label={disabledReason}>
      <span>{select}</span>
    </Tooltip>
  );
}

function optionText(l: Locale, translated: string[] | undefined): string {
  const name = l.name && l.name !== l.code ? ` · ${l.name}` : "";
  if (l.isDefault) return `${l.code}${name} (default)`;
  if (!translated) return `${l.code}${name}`;
  return `${l.code}${name}${translated.includes(l.code) ? " ✓" : " (not translated)"}`;
}

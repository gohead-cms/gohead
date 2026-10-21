import { createSystem, defaultConfig, defineConfig } from "@chakra-ui/react";

// Chakra v2's palettes: the admin UI was designed with them (v3's are more
// saturated, and its grays are neutral instead of slightly blue).
const v2Palettes = {
  gray: ["#F7FAFC", "#EDF2F7", "#E2E8F0", "#CBD5E0", "#A0AEC0", "#718096", "#4A5568", "#2D3748", "#1A202C", "#171923"],
  red: ["#FFF5F5", "#FED7D7", "#FEB2B2", "#FC8181", "#F56565", "#E53E3E", "#C53030", "#9B2C2C", "#822727", "#63171B"],
  orange: ["#FFFAF0", "#FEEBC8", "#FBD38D", "#F6AD55", "#ED8936", "#DD6B20", "#C05621", "#9C4221", "#7B341E", "#652B19"],
  yellow: ["#FFFFF0", "#FEFCBF", "#FAF089", "#F6E05E", "#ECC94B", "#D69E2E", "#B7791F", "#975A16", "#744210", "#5F370E"],
  green: ["#F0FFF4", "#C6F6D5", "#9AE6B4", "#68D391", "#48BB78", "#38A169", "#2F855A", "#276749", "#22543D", "#1C4532"],
  teal: ["#E6FFFA", "#B2F5EA", "#81E6D9", "#4FD1C5", "#38B2AC", "#319795", "#2C7A7B", "#285E61", "#234E52", "#1D4044"],
  blue: ["#EBF8FF", "#BEE3F8", "#90CDF4", "#63B3ED", "#4299E1", "#3182CE", "#2B6CB0", "#2C5282", "#2A4365", "#1A365D"],
  cyan: ["#EDFDFD", "#C4F1F9", "#9DECF9", "#76E4F7", "#0BC5EA", "#00B5D8", "#00A3C4", "#0987A0", "#086F83", "#065666"],
  purple: ["#FAF5FF", "#E9D8FD", "#D6BCFA", "#B794F4", "#9F7AEA", "#805AD5", "#6B46C1", "#553C9A", "#44337A", "#322659"],
  pink: ["#FFF5F7", "#FED7E2", "#FBB6CE", "#F687B3", "#ED64A6", "#D53F8C", "#B83280", "#97266D", "#702459", "#521B41"],
} as const;

const steps = ["50", "100", "200", "300", "400", "500", "600", "700", "800", "900"] as const;

const colors = Object.fromEntries(
  Object.entries(v2Palettes).map(([name, values]) => [
    name,
    Object.fromEntries(steps.map((step, i) => [step, { value: values[i] }])),
  ]),
);

// v2 filled solid buttons and badges with the 500 shade; v3 uses 600.
const solid = Object.fromEntries(
  Object.keys(v2Palettes)
    .filter((name) => name !== "gray")
    .map((name) => [name, { solid: { value: `{colors.${name}.500}` } }]),
);

/**
 * Chakra's default system, with the v2 details the admin UI was designed
 * with: v2 colors, uppercase badges and table column headers, v2's text
 * sizes in buttons, inputs, menus, form labels, help and error text, and
 * dialog and drawer titles, outlined badges and colored selected tabs.
 */
export const system = createSystem(
  defaultConfig,
  defineConfig({
    theme: {
      tokens: { colors },
      semanticTokens: { colors: solid },
      recipes: {
        badge: {
          base: { textTransform: "uppercase", fontWeight: "bold", letterSpacing: "wide" },
          // v2 outlined badges with the 500 shade, so the outline shows.
          variants: { variant: { outline: { color: "colorPalette.500", "--outline-shadow": "colors.colorPalette.500" } } },
        },
        // v2 text sizes: 14px at 32px high, 16px at 40px, 18px at 48px.
        button: { variants: { size: { xs: { textStyle: "sm" }, md: { textStyle: "md" }, lg: { textStyle: "lg" } } } },
        input: { variants: { size: { xs: { textStyle: "sm" }, md: { textStyle: "md" }, lg: { textStyle: "lg" } } } },
        textarea: { variants: { size: { md: { textStyle: "md" }, lg: { textStyle: "lg" } } } },
      },
      slotRecipes: {
        table: {
          slots: defaultConfig.theme?.slotRecipes?.table?.slots ?? [],
          base: {
            columnHeader: {
              textTransform: "uppercase",
              letterSpacing: "wider",
              fontSize: "xs",
              fontWeight: "bold",
              color: "fg.muted",
            },
          },
        },
        field: {
          slots: defaultConfig.theme?.slotRecipes?.field?.slots ?? [],
          base: {
            label: { textStyle: "md" },
            helperText: { textStyle: "sm" },
            errorText: { textStyle: "sm" },
          },
        },
        dialog: {
          slots: defaultConfig.theme?.slotRecipes?.dialog?.slots ?? [],
          base: { title: { textStyle: "xl" } },
        },
        drawer: {
          slots: defaultConfig.theme?.slotRecipes?.drawer?.slots ?? [],
          base: { title: { textStyle: "xl" } },
        },
        tabs: {
          slots: defaultConfig.theme?.slotRecipes?.tabs?.slots ?? [],
          // v2 colored the selected tab of the line variant, text and underline.
          variants: {
            variant: {
              line: {
                trigger: { _selected: { color: "colorPalette.600", "--indicator-color": "colors.colorPalette.600" } },
              },
            },
          },
        },
        menu: {
          slots: defaultConfig.theme?.slotRecipes?.menu?.slots ?? [],
          variants: { size: { md: { item: { textStyle: "md" } } } },
        },
        nativeSelect: {
          slots: defaultConfig.theme?.slotRecipes?.nativeSelect?.slots ?? [],
          variants: { size: { xs: { field: { textStyle: "sm" } }, md: { field: { textStyle: "md" } } } },
        },
      },
    },
  }),
);

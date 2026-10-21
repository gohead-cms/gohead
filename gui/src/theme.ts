import { createSystem, defaultConfig, defineConfig } from "@chakra-ui/react";

/**
 * Chakra's default system, with the v2 details the admin UI was designed
 * with: uppercase badges and table column headers.
 */
export const system = createSystem(
  defaultConfig,
  defineConfig({
    theme: {
      recipes: {
        badge: { base: { textTransform: "uppercase", fontWeight: "bold", letterSpacing: "wide" } },
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
      },
    },
  }),
);

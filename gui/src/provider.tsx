// src/components/ui/provider.tsx
import type { ReactNode } from "react";
import { ChakraProvider } from "@chakra-ui/react";
import { ThemeProvider } from "next-themes";

export function Provider({ children }: { children: ReactNode }) {
  return (
    <ChakraProvider>
      <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
        {children}
      </ThemeProvider>
    </ChakraProvider>
  );
}

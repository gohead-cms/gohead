// src/components/ui/provider.tsx
import { ChakraProvider } from "@chakra-ui/react";
import { ThemeProvider } from "next-themes";

export function Provider({ children }) {
  return (
    <ChakraProvider>
      <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
        {children}
      </ThemeProvider>
    </ChakraProvider>
  );
}
